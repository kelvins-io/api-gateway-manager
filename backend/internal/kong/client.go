package kongclient

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kong/go-kong/kong"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
)

type Client struct {
	kong *kong.Client
}

func New(adminAPI string) (*Client, error) {
	adminAPI = strings.TrimRight(adminAPI, "/")
	httpClient := &http.Client{Timeout: 15 * time.Second}
	kc, err := kong.NewClient(&adminAPI, httpClient)
	if err != nil {
		return nil, fmt.Errorf("create kong client: %w", err)
	}
	return &Client{kong: kc}, nil
}

// Probe checks whether Kong Admin API is reachable and healthy.
func Probe(ctx context.Context, adminAPI string) error {
	adminAPI = strings.TrimSpace(adminAPI)
	if adminAPI == "" {
		return fmt.Errorf("admin_api is empty")
	}
	client, err := New(adminAPI)
	if err != nil {
		return fmt.Errorf("admin api unreachable: %w", err)
	}
	if _, err := client.kong.Status(ctx); err != nil {
		return fmt.Errorf("admin api probe failed: %w", err)
	}
	return nil
}

type PublishResult struct {
	ServiceID string
	RouteID   string
}

func (c *Client) Publish(ctx context.Context, apiID uint64, snap model.APIConfigSnapshot, existingServiceID, existingRouteID string) (*PublishResult, error) {
	serviceName := fmt.Sprintf("agm-api-%d", apiID)
	host, port, path, protocol, err := parseUpstream(snap.UpstreamURL)
	if err != nil {
		return nil, err
	}

	svc := &kong.Service{
		Name:     kong.String(serviceName),
		Host:     kong.String(host),
		Port:     kong.Int(port),
		Path:     kong.String(path),
		Protocol: kong.String(protocol),
		Tags:     kong.StringSlice(fmt.Sprintf("agm-api-%d", apiID)),
	}

	var service *kong.Service
	if existingServiceID != "" {
		svc.ID = kong.String(existingServiceID)
		service, err = c.kong.Services.Update(ctx, svc)
		if err != nil {
			service, err = c.kong.Services.Create(ctx, svc)
		}
	} else {
		service, err = c.kong.Services.Create(ctx, svc)
	}
	if err != nil {
		return nil, fmt.Errorf("kong service upsert: %w", err)
	}

	methods := splitMethods(snap.Methods)
	paths := model.SplitPaths(snap.Path)
	if len(paths) == 0 {
		return nil, fmt.Errorf("at least one path is required")
	}
	routeName := fmt.Sprintf("agm-route-%d", apiID)
	route := &kong.Route{
		Name:      kong.String(routeName),
		Paths:     kong.StringSlice(paths...),
		Methods:   kong.StringSlice(methods...),
		StripPath: kong.Bool(snap.StripPath),
		Service:   &kong.Service{ID: service.ID},
		Tags:      kong.StringSlice(fmt.Sprintf("agm-api-%d", apiID)),
	}

	var createdRoute *kong.Route
	if existingRouteID != "" {
		route.ID = kong.String(existingRouteID)
		createdRoute, err = c.kong.Routes.Update(ctx, route)
		if err != nil {
			createdRoute, err = c.kong.Routes.Create(ctx, route)
		}
	} else {
		createdRoute, err = c.kong.Routes.Create(ctx, route)
	}
	if err != nil {
		return nil, fmt.Errorf("kong route upsert: %w", err)
	}

	return &PublishResult{
		ServiceID: *service.ID,
		RouteID:   *createdRoute.ID,
	}, nil
}

func (c *Client) Offline(ctx context.Context, serviceID, routeID string) error {
	if routeID != "" {
		if err := c.kong.Routes.Delete(ctx, &routeID); err != nil {
			// ignore not found
			if !isNotFound(err) {
				return fmt.Errorf("delete route: %w", err)
			}
		}
	}
	if serviceID != "" {
		if err := c.kong.Services.Delete(ctx, &serviceID); err != nil {
			if !isNotFound(err) {
				return fmt.Errorf("delete service: %w", err)
			}
		}
	}
	return nil
}

func splitMethods(methods string) []string {
	parts := strings.Split(methods, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToUpper(p))
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"GET"}
	}
	return out
}

func parseUpstream(raw string) (host string, port int, path string, protocol string, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", 0, "", "", fmt.Errorf("upstream_url is empty")
	}
	protocol = "http"
	if strings.HasPrefix(raw, "https://") {
		protocol = "https"
		raw = strings.TrimPrefix(raw, "https://")
	} else if strings.HasPrefix(raw, "http://") {
		raw = strings.TrimPrefix(raw, "http://")
	}

	path = "/"
	if idx := strings.Index(raw, "/"); idx >= 0 {
		path = raw[idx:]
		raw = raw[:idx]
	}

	port = 80
	if protocol == "https" {
		port = 443
	}
	host = raw
	if idx := strings.LastIndex(raw, ":"); idx >= 0 {
		host = raw[:idx]
		fmt.Sscanf(raw[idx+1:], "%d", &port)
	}
	if host == "" {
		return "", 0, "", "", fmt.Errorf("invalid upstream_url host")
	}
	return host, port, path, protocol, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "404")
}
