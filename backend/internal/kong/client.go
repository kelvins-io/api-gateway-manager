package kongclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"github.com/kong/go-kong/kong"
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

func (c *Client) Publish(ctx context.Context, apiID, spaceID, groupID uint64, snap model.APIConfigSnapshot, existingServiceID, existingRouteID string) (*PublishResult, error) {
	serviceName := fmt.Sprintf("agm-api-%d", apiID)
	host, port, path, protocol, retries, connTimeout, writeTimeout, readTimeout, err := serviceFields(snap)
	if err != nil {
		return nil, err
	}

	svc := &kong.Service{
		Name:           kong.String(serviceName),
		Host:           kong.String(host),
		Port:           kong.Int(port),
		Path:           kong.String(path),
		Protocol:       kong.String(protocol),
		Retries:        kong.Int(retries),
		ConnectTimeout: kong.Int(connTimeout),
		WriteTimeout:   kong.Int(writeTimeout),
		ReadTimeout:    kong.Int(readTimeout),
		Tags:           kong.StringSlice(fmt.Sprintf("agm-api-%d", apiID)),
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

	methods := splitMethods(firstNonEmpty(snap.AccessMethods, snap.LegacyMethods))
	paths := model.SplitPaths(firstNonEmpty(snap.AccessPath, snap.LegacyPath))
	if len(paths) == 0 {
		return nil, fmt.Errorf("at least one path is required")
	}
	routeName := model.KongRouteName(spaceID, groupID, apiID)
	route := &kong.Route{
		Name:              kong.String(routeName),
		Paths:             kong.StringSlice(paths...),
		Methods:           kong.StringSlice(methods...),
		Protocols:         kong.StringSlice(accessProtocols(snap.AccessProtocols)...),
		StripPath:         kong.Bool(snap.EffectiveStripPath()),
		RequestBuffering:  kong.Bool(snap.EffectiveRequestBuffering()),
		ResponseBuffering: kong.Bool(snap.EffectiveResponseBuffering()),
		Service:           &kong.Service{ID: service.ID},
		Tags:              kong.StringSlice(fmt.Sprintf("agm-api-%d", apiID)),
	}
	if hosts := splitAccessHosts(snap.AccessHosts); len(hosts) > 0 {
		route.Hosts = kong.StringSlice(hosts...)
	}
	if len(snap.AccessHeaders) > 0 {
		route.Headers = snap.AccessHeaders
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func splitAccessHosts(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	})
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func accessProtocols(raw string) []string {
	allowed := map[string]struct{}{"http": {}, "https": {}, "grpc": {}, "grpcs": {}}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if _, ok := allowed[p]; !ok {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{"http"}
	}
	return out
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

func serviceFields(snap model.APIConfigSnapshot) (host string, port int, path, protocol string, retries, connTimeout, writeTimeout, readTimeout int, err error) {
	if snap.EffectiveProtocol() == "" && snap.KongHost == "" && snap.EffectiveHost() == "" {
		host, port, path, protocol, err = parseUpstream(snap.UpstreamURL)
		if err != nil {
			return
		}
		return host, port, path, protocol, 5, 60000, 60000, 60000, nil
	}
	protocol = snap.EffectiveProtocol()
	if protocol == "" {
		protocol = "http"
	}
	host = snap.KongHost
	if host == "" {
		host = snap.EffectiveHost()
	}
	port = snap.EffectivePort()
	if port == 0 {
		port = 80
	}
	path = snap.ServicePath
	if path == "" {
		path = "/"
	}
	retries = snap.EffectiveRetries()
	connTimeout = snap.EffectiveConnectTimeout()
	writeTimeout = snap.EffectiveWriteTimeout()
	readTimeout = snap.EffectiveReadTimeout()
	if host == "" {
		err = fmt.Errorf("service host is empty")
	}
	return
}

type TargetSync struct {
	Target string
	Weight int
}

type UpstreamSync struct {
	KongName               string
	ExistingID             string
	Algorithm              string
	Slots                  int
	HashOn                 string
	HashFallback           string
	HashOnHeader           string
	HashFallbackHeader     string
	HashOnCookie           string
	HashOnCookiePath       string
	HashOnQueryArg         string
	HashFallbackQueryArg   string
	HashOnURICapture       string
	HashFallbackURICapture string
	Healthchecks           json.RawMessage
	Targets                []TargetSync
}

func (c *Client) SyncUpstream(ctx context.Context, in UpstreamSync) (string, error) {
	hc, err := decodeHealthchecks(in.Healthchecks)
	if err != nil {
		return "", err
	}
	slots := in.Slots
	if slots <= 0 {
		slots = 10000
	}
	algo := in.Algorithm
	if algo == "" {
		algo = "round-robin"
	}
	up := &kong.Upstream{
		Name:                   kong.String(in.KongName),
		Algorithm:              kong.String(algo),
		Slots:                  kong.Int(slots),
		Healthchecks:           hc,
		HashOn:                 emptyStringPtr(in.HashOn),
		HashFallback:           emptyStringPtr(in.HashFallback),
		HashOnHeader:           emptyStringPtr(in.HashOnHeader),
		HashFallbackHeader:     emptyStringPtr(in.HashFallbackHeader),
		HashOnCookie:           emptyStringPtr(in.HashOnCookie),
		HashOnCookiePath:       emptyStringPtr(in.HashOnCookiePath),
		HashOnQueryArg:         emptyStringPtr(in.HashOnQueryArg),
		HashFallbackQueryArg:   emptyStringPtr(in.HashFallbackQueryArg),
		HashOnURICapture:       emptyStringPtr(in.HashOnURICapture),
		HashFallbackURICapture: emptyStringPtr(in.HashFallbackURICapture),
	}
	var saved *kong.Upstream
	if in.ExistingID != "" {
		up.ID = kong.String(in.ExistingID)
		saved, err = c.kong.Upstreams.Update(ctx, up)
		if err != nil {
			up.ID = nil
			saved, err = c.kong.Upstreams.Create(ctx, up)
		}
	} else {
		saved, err = c.kong.Upstreams.Create(ctx, up)
		if err != nil {
			existing, getErr := c.kong.Upstreams.Get(ctx, &in.KongName)
			if getErr != nil {
				return "", fmt.Errorf("kong upstream upsert: %w", err)
			}
			up.ID = existing.ID
			saved, err = c.kong.Upstreams.Update(ctx, up)
		}
	}
	if err != nil {
		return "", fmt.Errorf("kong upstream upsert: %w", err)
	}
	if err := c.replaceTargets(ctx, *saved.ID, in.Targets); err != nil {
		return "", err
	}
	return *saved.ID, nil
}

func (c *Client) DeleteUpstream(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	if err := c.kong.Upstreams.Delete(ctx, &id); err != nil && !isNotFound(err) {
		return fmt.Errorf("delete upstream: %w", err)
	}
	return nil
}

func (c *Client) replaceTargets(ctx context.Context, upstreamID string, targets []TargetSync) error {
	existing, err := c.kong.Targets.ListAll(ctx, &upstreamID)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("list targets: %w", err)
	}
	for _, t := range existing {
		if t.ID == nil {
			continue
		}
		if err := c.kong.Targets.Delete(ctx, &upstreamID, t.ID); err != nil && !isNotFound(err) {
			return fmt.Errorf("delete target: %w", err)
		}
	}
	for _, t := range targets {
		weight := t.Weight
		if weight <= 0 {
			weight = 100
		}
		if _, err := c.kong.Targets.Create(ctx, &upstreamID, &kong.Target{
			Target: kong.String(t.Target),
			Weight: kong.Int(weight),
		}); err != nil {
			return fmt.Errorf("create target: %w", err)
		}
	}
	return nil
}

func decodeHealthchecks(raw json.RawMessage) (*kong.Healthcheck, error) {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return nil, nil
	}
	var hc kong.Healthcheck
	if err := json.Unmarshal(raw, &hc); err != nil {
		return nil, fmt.Errorf("healthchecks: %w", err)
	}
	return &hc, nil
}

func emptyStringPtr(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return kong.String(s)
}

type CredentialSync struct {
	Plugin string
	Config map[string]string
}

type ConsumerSync struct {
	KongUsername string
	KongCustomID string
	ExistingID   string
	Tag          string
	Credentials  []CredentialSync
}

func (c *Client) SyncConsumer(ctx context.Context, in ConsumerSync) (string, error) {
	body := &kong.Consumer{
		Username: kong.String(in.KongUsername),
		Tags:     kong.StringSlice(in.Tag),
	}
	if in.KongCustomID != "" {
		body.CustomID = kong.String(in.KongCustomID)
	}
	var saved *kong.Consumer
	var err error
	if in.ExistingID != "" {
		body.ID = kong.String(in.ExistingID)
		saved, err = c.kong.Consumers.Update(ctx, body)
		if err != nil {
			body.ID = nil
			saved, err = c.kong.Consumers.Create(ctx, body)
		}
	} else {
		saved, err = c.kong.Consumers.Create(ctx, body)
		if err != nil {
			existing, getErr := c.kong.Consumers.Get(ctx, &in.KongUsername)
			if getErr != nil {
				return "", fmt.Errorf("kong consumer upsert: %w", err)
			}
			body.ID = existing.ID
			saved, err = c.kong.Consumers.Update(ctx, body)
		}
	}
	if err != nil {
		return "", fmt.Errorf("kong consumer upsert: %w", err)
	}
	if err := c.replaceCredentials(ctx, *saved.ID, in.Credentials); err != nil {
		return "", err
	}
	return *saved.ID, nil
}

func (c *Client) DeleteConsumer(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	if err := c.kong.Consumers.Delete(ctx, &id); err != nil && !isNotFound(err) {
		return fmt.Errorf("delete consumer: %w", err)
	}
	return nil
}

func (c *Client) replaceCredentials(ctx context.Context, consumerID string, creds []CredentialSync) error {
	if err := c.clearCredentials(ctx, consumerID); err != nil {
		return err
	}
	for _, cred := range creds {
		if err := c.createCredential(ctx, consumerID, cred); err != nil {
			return fmt.Errorf("create %s credential: %w", cred.Plugin, err)
		}
	}
	return nil
}

func (c *Client) clearCredentials(ctx context.Context, consumerID string) error {
	keys, _, err := c.kong.KeyAuths.ListForConsumer(ctx, &consumerID, &kong.ListOpt{Size: 1000})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("list key-auth: %w", err)
	}
	for _, item := range keys {
		if item.ID == nil {
			continue
		}
		if err := c.kong.KeyAuths.Delete(ctx, &consumerID, item.ID); err != nil && !isNotFound(err) {
			return err
		}
	}
	basics, _, err := c.kong.BasicAuths.ListForConsumer(ctx, &consumerID, &kong.ListOpt{Size: 1000})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("list basic-auth: %w", err)
	}
	for _, item := range basics {
		if item.ID == nil {
			continue
		}
		if err := c.kong.BasicAuths.Delete(ctx, &consumerID, item.ID); err != nil && !isNotFound(err) {
			return err
		}
	}
	jwts, _, err := c.kong.JWTAuths.ListForConsumer(ctx, &consumerID, &kong.ListOpt{Size: 1000})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("list jwt: %w", err)
	}
	for _, item := range jwts {
		if item.ID == nil {
			continue
		}
		if err := c.kong.JWTAuths.Delete(ctx, &consumerID, item.ID); err != nil && !isNotFound(err) {
			return err
		}
	}
	hmacs, _, err := c.kong.HMACAuths.ListForConsumer(ctx, &consumerID, &kong.ListOpt{Size: 1000})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("list hmac-auth: %w", err)
	}
	for _, item := range hmacs {
		if item.ID == nil {
			continue
		}
		if err := c.kong.HMACAuths.Delete(ctx, &consumerID, item.ID); err != nil && !isNotFound(err) {
			return err
		}
	}
	acls, _, err := c.kong.ACLs.ListForConsumer(ctx, &consumerID, &kong.ListOpt{Size: 1000})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("list acl: %w", err)
	}
	for _, item := range acls {
		if item.ID == nil {
			continue
		}
		if err := c.kong.ACLs.Delete(ctx, &consumerID, item.ID); err != nil && !isNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *Client) createCredential(ctx context.Context, consumerID string, cred CredentialSync) error {
	switch cred.Plugin {
	case "key-auth":
		_, err := c.kong.KeyAuths.Create(ctx, &consumerID, &kong.KeyAuth{Key: kong.String(cred.Config["key"])})
		return err
	case "basic-auth":
		_, err := c.kong.BasicAuths.Create(ctx, &consumerID, &kong.BasicAuth{
			Username: kong.String(cred.Config["username"]),
			Password: kong.String(cred.Config["password"]),
		})
		return err
	case "jwt":
		body := &kong.JWTAuth{
			Algorithm: kong.String(cred.Config["algorithm"]),
		}
		if cred.Config["key"] != "" {
			body.Key = kong.String(cred.Config["key"])
		}
		if cred.Config["secret"] != "" {
			body.Secret = kong.String(cred.Config["secret"])
		}
		if cred.Config["rsa_public_key"] != "" {
			body.RSAPublicKey = kong.String(cred.Config["rsa_public_key"])
		}
		_, err := c.kong.JWTAuths.Create(ctx, &consumerID, body)
		return err
	case "hmac-auth":
		_, err := c.kong.HMACAuths.Create(ctx, &consumerID, &kong.HMACAuth{
			Username: kong.String(cred.Config["username"]),
			Secret:   kong.String(cred.Config["secret"]),
		})
		return err
	case "acl":
		_, err := c.kong.ACLs.Create(ctx, &consumerID, &kong.ACLGroup{Group: kong.String(cred.Config["group"])})
		return err
	default:
		return fmt.Errorf("unsupported plugin %s", cred.Plugin)
	}
}

type PluginSync struct {
	Name         string
	InstanceName string
	Config       map[string]interface{}
	Enabled      bool
	Tag          string
}

// ReplaceServicePlugins removes plugins previously managed for this API and creates the given set on the service.
func (c *Client) ReplaceServicePlugins(ctx context.Context, serviceID, managedTag string, plugins []PluginSync) error {
	if serviceID == "" {
		return nil
	}
	existing, err := c.kong.Plugins.ListAllForService(ctx, &serviceID)
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("list service plugins: %w", err)
	}
	for _, item := range existing {
		if item == nil || item.ID == nil || !hasTag(item.Tags, managedTag) {
			continue
		}
		if err := c.kong.Plugins.Delete(ctx, item.ID); err != nil && !isNotFound(err) {
			return fmt.Errorf("delete plugin: %w", err)
		}
	}
	for _, p := range plugins {
		body := &kong.Plugin{
			Name:         kong.String(p.Name),
			InstanceName: kong.String(p.InstanceName),
			Config:       kong.Configuration(p.Config),
			Enabled:      kong.Bool(p.Enabled),
			Service:      &kong.Service{ID: kong.String(serviceID)},
			Tags:         kong.StringSlice(managedTag, p.Tag),
		}
		if _, err := c.kong.Plugins.Create(ctx, body); err != nil {
			return fmt.Errorf("create plugin %s: %w", p.Name, err)
		}
	}
	return nil
}

func hasTag(tags []*string, want string) bool {
	for _, t := range tags {
		if t != nil && *t == want {
			return true
		}
	}
	return false
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "404")
}
