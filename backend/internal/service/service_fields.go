package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	kongclient "github.com/kelvins-io/api-gateway-manager/internal/kong"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
)

const (
	defaultRetries = 5
	defaultTimeout = 60000
)

var hostPattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?$|^(\d{1,3}\.){3}\d{1,3}$`)

type serviceInput struct {
	Protocol       string
	HostKind       string
	Host           string
	UpstreamID     *uint64
	Port           int
	ServicePath    string
	Retries        *int
	ConnectTimeout *int
	WriteTimeout   *int
	ReadTimeout    *int
}

type serviceFields struct {
	Protocol       string
	HostKind       string
	Host           string
	UpstreamID     *uint64
	Port           int
	ServicePath    string
	Retries        int
	ConnectTimeout int
	WriteTimeout   int
	ReadTimeout    int
}

func (s *APIService) normalizeService(spaceID uint64, in serviceInput, create bool) (serviceFields, error) {
	out := serviceFields{
		Protocol:       strings.ToLower(strings.TrimSpace(in.Protocol)),
		HostKind:       strings.TrimSpace(in.HostKind),
		Host:           strings.TrimSpace(in.Host),
		UpstreamID:     in.UpstreamID,
		Port:           in.Port,
		ServicePath:    strings.TrimSpace(in.ServicePath),
		Retries:        defaultRetries,
		ConnectTimeout: defaultTimeout,
		WriteTimeout:   defaultTimeout,
		ReadTimeout:    defaultTimeout,
	}
	if out.Protocol == "" {
		out.Protocol = "http"
	}
	switch out.Protocol {
	case "http", "https", "grpc", "grpcs":
	default:
		return out, fmt.Errorf("%w: protocol must be http, https, grpc or grpcs", ErrBadRequest)
	}
	if out.HostKind == "" {
		out.HostKind = model.HostKindDirect
	}
	if out.HostKind != model.HostKindDirect && out.HostKind != model.HostKindUpstream {
		return out, fmt.Errorf("%w: host_kind must be direct or upstream", ErrBadRequest)
	}
	if out.Port == 0 && create {
		out.Port = 80
	}
	if out.Port < 80 || out.Port > 65535 {
		return out, fmt.Errorf("%w: port must be between 80 and 65535", ErrBadRequest)
	}
	if out.ServicePath == "" {
		out.ServicePath = "/"
	}
	if strings.ContainsAny(out.ServicePath, ",;\n") || !strings.HasPrefix(out.ServicePath, "/") {
		return out, fmt.Errorf("%w: service_path must be a single path starting with /", ErrBadRequest)
	}
	if in.Retries != nil {
		if *in.Retries < 0 {
			return out, fmt.Errorf("%w: retries must be >= 0", ErrBadRequest)
		}
		out.Retries = *in.Retries
	}
	if err := applyTimeout(&out.ConnectTimeout, in.ConnectTimeout, "connect_timeout"); err != nil {
		return out, err
	}
	if err := applyTimeout(&out.WriteTimeout, in.WriteTimeout, "write_timeout"); err != nil {
		return out, err
	}
	if err := applyTimeout(&out.ReadTimeout, in.ReadTimeout, "read_timeout"); err != nil {
		return out, err
	}
	if out.HostKind == model.HostKindUpstream {
		if out.UpstreamID == nil || *out.UpstreamID == 0 {
			return out, fmt.Errorf("%w: upstream_id is required", ErrBadRequest)
		}
		var up model.Upstream
		if err := s.db.Where("id = ? AND space_id = ?", *out.UpstreamID, spaceID).First(&up).Error; err != nil {
			return out, fmt.Errorf("%w: upstream not found in this space", ErrBadRequest)
		}
		out.Host = up.Name
		return out, nil
	}
	out.UpstreamID = nil
	if out.Host == "" || !hostPattern.MatchString(out.Host) {
		return out, fmt.Errorf("%w: host must be an IP or domain", ErrBadRequest)
	}
	return out, nil
}

func applyTimeout(dst *int, src *int, name string) error {
	if src == nil {
		return nil
	}
	if *src < 0 {
		return fmt.Errorf("%w: %s must be >= 0", ErrBadRequest, name)
	}
	*dst = *src
	return nil
}

func (s *APIService) snapshotOf(ctx context.Context, api *model.API) (model.APIConfigSnapshot, error) {
	snap := model.APIConfigSnapshot{
		Name:                  api.Name,
		AccessPath:            api.AccessPath,
		AccessMethods:         api.AccessMethods,
		AccessProtocols:       api.AccessProtocols,
		AccessHosts:           api.AccessHosts,
		AccessHeaders:         decodeHeaders(api.AccessHeaders),
		UpstreamURL:           api.UpstreamURL,
		ServiceProtocol:       api.ServiceProtocol,
		ServiceHostKind:       api.ServiceHostKind,
		ServiceHost:           api.ServiceHost,
		ServicePort:           api.ServicePort,
		ServicePath:           api.ServicePath,
		ServiceRetries:        api.ServiceRetries,
		ServiceConnectTimeout: api.ServiceConnectTimeout,
		ServiceWriteTimeout:   api.ServiceWriteTimeout,
		ServiceReadTimeout:    api.ServiceReadTimeout,
		AccessStripPath:       api.AccessStripPath,
		KongHost:              api.ServiceHost,
	}
	if api.ServiceUpstreamID != nil {
		snap.ServiceUpstreamID = *api.ServiceUpstreamID
	}
	if snap.ServiceProtocol == "" && snap.ServiceHost == "" && api.UpstreamURL != "" {
		return snap, nil
	}
	if api.ServiceHostKind == model.HostKindUpstream {
		if api.ServiceUpstreamID == nil || api.Group == nil || api.Group.Gateway == nil {
			return snap, fmt.Errorf("%w: upstream is required", ErrBadRequest)
		}
		kongHost, err := s.syncUpstreamGateway(ctx, *api.ServiceUpstreamID, api.Group.Gateway)
		if err != nil {
			return snap, err
		}
		snap.KongHost = kongHost
	}
	return snap, nil
}

func (s *APIService) syncUpstreamGateway(ctx context.Context, upstreamID uint64, gw *model.Gateway) (string, error) {
	var up model.Upstream
	if err := s.db.Preload("Targets").First(&up, upstreamID).Error; err != nil {
		return "", fmt.Errorf("%w: upstream not found", ErrNotFound)
	}
	var binding model.UpstreamGateway
	_ = s.db.Where("upstream_id = ? AND gateway_id = ?", up.ID, gw.ID).First(&binding).Error
	client, err := kongclient.New(gw.AdminAPI)
	if err != nil {
		return "", err
	}
	kongID, err := client.SyncUpstream(ctx, upstreamToSync(up, binding.KongUpstreamID))
	if err != nil {
		return "", err
	}
	binding.UpstreamID = up.ID
	binding.GatewayID = gw.ID
	binding.KongUpstreamID = kongID
	if binding.ID == 0 {
		if err := s.db.Create(&binding).Error; err != nil {
			return "", err
		}
	} else if err := s.db.Save(&binding).Error; err != nil {
		return "", err
	}
	return model.KongUpstreamName(up.SpaceID, up.Name), nil
}

func nilIfZero(id uint64) interface{} {
	if id == 0 {
		return nil
	}
	return id
}
