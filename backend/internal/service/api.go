package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	kongclient "github.com/kelvins-io/api-gateway-manager/internal/kong"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type APIService struct {
	db *gorm.DB
}

func NewAPIService(db *gorm.DB) *APIService {
	return &APIService{db: db}
}

type CreateAPIInput struct {
	Name                  string              `json:"name" binding:"required,min=1,max=128"`
	AccessPath            string              `json:"access_path" binding:"required,min=1,max=2048"`
	AccessMethods         string              `json:"access_methods" binding:"required"`
	AccessProtocols       string              `json:"access_protocols"`
	AccessHosts           string              `json:"access_hosts"`
	AccessHeaders         map[string][]string `json:"access_headers"`
	ServiceProtocol       string              `json:"service_protocol"`
	ServiceHostKind       string              `json:"service_host_kind"`
	ServiceHost           string              `json:"service_host"`
	ServiceUpstreamID     *uint64             `json:"service_upstream_id"`
	ServicePort           int                 `json:"service_port"`
	ServicePath           string              `json:"service_path"`
	ServiceRetries        *int                `json:"service_retries"`
	ServiceConnectTimeout *int                `json:"service_connect_timeout"`
	ServiceWriteTimeout   *int                `json:"service_write_timeout"`
	ServiceReadTimeout    *int                `json:"service_read_timeout"`
	AccessStripPath       *bool               `json:"access_strip_path"`
}

type UpdateAPIInput struct {
	Name                  string              `json:"name" binding:"omitempty,min=1,max=128"`
	AccessPath            string              `json:"access_path" binding:"omitempty,min=1,max=2048"`
	AccessMethods         string              `json:"access_methods"`
	AccessProtocols       string              `json:"access_protocols"`
	AccessHosts           string              `json:"access_hosts"`
	AccessHeaders         map[string][]string `json:"access_headers"`
	ServiceProtocol       string              `json:"service_protocol"`
	ServiceHostKind       string              `json:"service_host_kind"`
	ServiceHost           string              `json:"service_host"`
	ServiceUpstreamID     *uint64             `json:"service_upstream_id"`
	ServicePort           *int                `json:"service_port"`
	ServicePath           string              `json:"service_path"`
	ServiceRetries        *int                `json:"service_retries"`
	ServiceConnectTimeout *int                `json:"service_connect_timeout"`
	ServiceWriteTimeout   *int                `json:"service_write_timeout"`
	ServiceReadTimeout    *int                `json:"service_read_timeout"`
	AccessStripPath       *bool               `json:"access_strip_path"`
}

type SwitchVersionInput struct {
	Version string `json:"version" binding:"required"`
}

func (s *APIService) Create(groupID uint64, in CreateAPIInput) (*model.API, error) {
	group, err := s.getGroup(groupID)
	if err != nil {
		return nil, err
	}
	strip := true
	if in.AccessStripPath != nil {
		strip = *in.AccessStripPath
	}
	paths := model.SplitPaths(in.AccessPath)
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: at least one path is required", ErrBadRequest)
	}
	paths, err = s.withSpacePrefix(group.SpaceID, paths)
	if err != nil {
		return nil, err
	}
	accessHosts, err := normalizeAccessHosts(in.AccessHosts)
	if err != nil {
		return nil, err
	}
	accessHeaders, err := normalizeAccessHeaders(in.AccessHeaders)
	if err != nil {
		return nil, err
	}
	svcFields, err := s.normalizeService(group.SpaceID, serviceInput{
		Protocol: in.ServiceProtocol, HostKind: in.ServiceHostKind, Host: in.ServiceHost, UpstreamID: in.ServiceUpstreamID,
		Port: in.ServicePort, ServicePath: in.ServicePath, Retries: in.ServiceRetries,
		ConnectTimeout: in.ServiceConnectTimeout, WriteTimeout: in.ServiceWriteTimeout, ReadTimeout: in.ServiceReadTimeout,
	}, true)
	if err != nil {
		return nil, err
	}
	api := &model.API{
		GroupID:               groupID,
		Name:                  in.Name,
		AccessPath:            strings.Join(paths, ","),
		AccessMethods:         normalizeMethods(in.AccessMethods),
		AccessProtocols:       normalizeAccessProtocols(in.AccessProtocols),
		AccessHosts:           accessHosts,
		AccessHeaders:         accessHeaders,
		ServiceProtocol:       svcFields.Protocol,
		ServiceHostKind:       svcFields.HostKind,
		ServiceHost:           svcFields.Host,
		ServiceUpstreamID:     svcFields.UpstreamID,
		ServicePort:           svcFields.Port,
		ServicePath:           svcFields.ServicePath,
		ServiceRetries:        svcFields.Retries,
		ServiceConnectTimeout: svcFields.ConnectTimeout,
		ServiceWriteTimeout:   svcFields.WriteTimeout,
		ServiceReadTimeout:    svcFields.ReadTimeout,
		AccessStripPath:       strip,
		Status:                model.APIStatusDraft,
	}
	if err := s.db.Create(api).Error; err != nil {
		return nil, err
	}
	return s.Get(api.ID)
}

func (s *APIService) ListByGroup(groupID uint64) ([]model.API, error) {
	var list []model.API
	err := s.db.Preload("Group").Preload("Group.Gateway").Preload("Group.Space").Where("group_id = ?", groupID).Order("id desc").Find(&list).Error
	return list, err
}

func (s *APIService) Get(id uint64) (*model.API, error) {
	var api model.API
	if err := s.db.Preload("Group").Preload("Group.Gateway").First(&api, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &api, nil
}

func (s *APIService) Update(id uint64, in UpdateAPIInput) (*model.API, error) {
	api, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	updates := map[string]interface{}{}
	if in.Name != "" {
		updates["name"] = in.Name
	}
	if in.AccessPath != "" {
		paths := model.SplitPaths(in.AccessPath)
		if len(paths) == 0 {
			return nil, fmt.Errorf("%w: at least one path is required", ErrBadRequest)
		}
		if api.Group == nil {
			return nil, ErrNotFound
		}
		paths, err = s.withSpacePrefix(api.Group.SpaceID, paths)
		if err != nil {
			return nil, err
		}
		updates["access_path"] = strings.Join(paths, ",")
	}
	if in.AccessMethods != "" {
		updates["access_methods"] = normalizeMethods(in.AccessMethods)
	}
	if in.AccessProtocols != "" {
		updates["access_protocols"] = normalizeAccessProtocols(in.AccessProtocols)
	}
	hosts, err := normalizeAccessHosts(in.AccessHosts)
	if err != nil {
		return nil, err
	}
	headers, err := normalizeAccessHeaders(in.AccessHeaders)
	if err != nil {
		return nil, err
	}
	updates["access_hosts"] = hosts
	updates["access_headers"] = headers
	if in.ServiceProtocol != "" || in.ServiceHostKind != "" || in.ServiceHost != "" || in.ServiceUpstreamID != nil || in.ServicePort != nil || in.ServicePath != "" || in.ServiceRetries != nil || in.ServiceConnectTimeout != nil || in.ServiceWriteTimeout != nil || in.ServiceReadTimeout != nil {
		port := api.ServicePort
		if in.ServicePort != nil {
			port = *in.ServicePort
		}
		svcFields, err := s.normalizeService(api.Group.SpaceID, serviceInput{
			Protocol: in.ServiceProtocol, HostKind: in.ServiceHostKind, Host: in.ServiceHost, UpstreamID: in.ServiceUpstreamID,
			Port: port, ServicePath: in.ServicePath, Retries: in.ServiceRetries,
			ConnectTimeout: in.ServiceConnectTimeout, WriteTimeout: in.ServiceWriteTimeout, ReadTimeout: in.ServiceReadTimeout,
		}, false)
		if err != nil {
			return nil, err
		}
		if svcFields.Protocol == "" {
			svcFields.Protocol = api.ServiceProtocol
		}
		if svcFields.HostKind == "" {
			svcFields.HostKind = api.ServiceHostKind
		}
		if svcFields.ServicePath == "" {
			svcFields.ServicePath = api.ServicePath
		}
		updates["service_protocol"] = svcFields.Protocol
		updates["service_host_kind"] = svcFields.HostKind
		updates["service_host"] = svcFields.Host
		if svcFields.UpstreamID == nil {
			updates["service_upstream_id"] = gorm.Expr("NULL")
		} else {
			updates["service_upstream_id"] = *svcFields.UpstreamID
		}
		updates["service_port"] = svcFields.Port
		updates["service_path"] = svcFields.ServicePath
		updates["service_retries"] = svcFields.Retries
		updates["service_connect_timeout"] = svcFields.ConnectTimeout
		updates["service_write_timeout"] = svcFields.WriteTimeout
		updates["service_read_timeout"] = svcFields.ReadTimeout
	}
	if in.AccessStripPath != nil {
		updates["access_strip_path"] = *in.AccessStripPath
	}
	if len(updates) > 0 {
		if err := s.db.Model(api).Updates(updates).Error; err != nil {
			return nil, err
		}
	}
	return s.Get(id)
}

func (s *APIService) Delete(id uint64) error {
	api, err := s.Get(id)
	if err != nil {
		return err
	}
	if api.Status == model.APIStatusPublished {
		return fmt.Errorf("%w: please offline api before delete", ErrConflict)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("api_id = ?", id).Delete(&model.APIVersion{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.API{}, id).Error
	})
}

func (s *APIService) Publish(ctx context.Context, id uint64) (*model.API, error) {
	api, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if api.Group == nil || api.Group.Gateway == nil {
		return nil, fmt.Errorf("%w: gateway not configured", ErrBadRequest)
	}

	snap, err := s.snapshotOf(ctx, api)
	if err != nil {
		return nil, err
	}

	client, err := kongclient.New(api.Group.Gateway.AdminAPI)
	if err != nil {
		return nil, err
	}
	result, err := client.Publish(ctx, api.ID, snap, api.KongServiceID, api.KongRouteID)
	if err != nil {
		return nil, fmt.Errorf("publish to kong: %w", err)
	}

	version, err := s.nextVersion(api.ID)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(snap)
	if err != nil {
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		ver := &model.APIVersion{
			APIID:          api.ID,
			Version:        version,
			ConfigSnapshot: datatypes.JSON(raw),
			PublishedAt:    time.Now(),
		}
		if err := tx.Create(ver).Error; err != nil {
			return err
		}
		return tx.Model(api).Updates(map[string]interface{}{
			"status":          model.APIStatusPublished,
			"current_version": version,
			"kong_service_id": result.ServiceID,
			"kong_route_id":   result.RouteID,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *APIService) Offline(ctx context.Context, id uint64) (*model.API, error) {
	api, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if api.Status != model.APIStatusPublished {
		return nil, fmt.Errorf("%w: api is not published", ErrBadRequest)
	}
	if api.Group == nil || api.Group.Gateway == nil {
		return nil, fmt.Errorf("%w: gateway not configured", ErrBadRequest)
	}

	client, err := kongclient.New(api.Group.Gateway.AdminAPI)
	if err != nil {
		return nil, err
	}
	if err := client.Offline(ctx, api.KongServiceID, api.KongRouteID); err != nil {
		return nil, fmt.Errorf("offline from kong: %w", err)
	}

	if err := s.db.Model(api).Updates(map[string]interface{}{
		"status":          model.APIStatusOffline,
		"kong_service_id": "",
		"kong_route_id":   "",
	}).Error; err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *APIService) SwitchVersion(ctx context.Context, id uint64, version string) (*model.API, error) {
	api, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if api.Group == nil || api.Group.Gateway == nil {
		return nil, fmt.Errorf("%w: gateway not configured", ErrBadRequest)
	}

	var ver model.APIVersion
	if err := s.db.Where("api_id = ? AND version = ?", id, version).First(&ver).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: version not found", ErrNotFound)
		}
		return nil, err
	}

	var snap model.APIConfigSnapshot
	if err := json.Unmarshal(ver.ConfigSnapshot, &snap); err != nil {
		return nil, err
	}
	if snap.AccessPath == "" {
		snap.AccessPath = snap.LegacyPath
	}
	if snap.AccessMethods == "" {
		snap.AccessMethods = snap.LegacyMethods
	}
	snap.AccessStripPath = snap.EffectiveStripPath()
	snap.LegacyStripPath = nil
	snap.ServiceProtocol = snap.EffectiveProtocol()
	snap.ServiceHostKind = snap.EffectiveHostKind()
	snap.ServiceHost = snap.EffectiveHost()
	snap.ServiceUpstreamID = snap.EffectiveUpstreamID()
	snap.ServicePort = snap.EffectivePort()
	snap.ServiceRetries = snap.EffectiveRetries()
	snap.ServiceConnectTimeout = snap.EffectiveConnectTimeout()
	snap.ServiceWriteTimeout = snap.EffectiveWriteTimeout()
	snap.ServiceReadTimeout = snap.EffectiveReadTimeout()
	snap.LegacyProtocol = ""
	snap.LegacyHostKind = ""
	snap.LegacyHost = ""
	snap.LegacyUpstreamID = nil
	snap.LegacyPort = nil
	snap.LegacyRetries = nil
	snap.LegacyConnectTimeout = nil
	snap.LegacyWriteTimeout = nil
	snap.LegacyReadTimeout = nil

	if snap.ServiceHostKind == model.HostKindUpstream && snap.ServiceUpstreamID > 0 {
		kongHost, syncErr := s.syncUpstreamGateway(ctx, snap.ServiceUpstreamID, api.Group.Gateway)
		if syncErr != nil {
			return nil, syncErr
		}
		snap.KongHost = kongHost
	}

	client, err := kongclient.New(api.Group.Gateway.AdminAPI)
	if err != nil {
		return nil, err
	}

	// offline current if published
	if api.Status == model.APIStatusPublished {
		if err := client.Offline(ctx, api.KongServiceID, api.KongRouteID); err != nil {
			return nil, fmt.Errorf("offline current version: %w", err)
		}
	}

	result, err := client.Publish(ctx, api.ID, snap, "", "")
	if err != nil {
		return nil, fmt.Errorf("publish selected version: %w", err)
	}

	if err := s.db.Model(api).Updates(map[string]interface{}{
		"name":                    snap.Name,
		"access_path":             snap.AccessPath,
		"access_methods":          snap.AccessMethods,
		"access_protocols":        snap.AccessProtocols,
		"access_hosts":            snap.AccessHosts,
		"access_headers":          mustHeaderJSON(snap.AccessHeaders),
		"upstream_url":            snap.UpstreamURL,
		"service_protocol":        snap.ServiceProtocol,
		"service_host_kind":       snap.ServiceHostKind,
		"service_host":            snap.ServiceHost,
		"service_upstream_id":     nilIfZero(snap.ServiceUpstreamID),
		"service_port":            snap.ServicePort,
		"service_path":            snap.ServicePath,
		"service_retries":         snap.ServiceRetries,
		"service_connect_timeout": snap.ServiceConnectTimeout,
		"service_write_timeout":   snap.ServiceWriteTimeout,
		"service_read_timeout":    snap.ServiceReadTimeout,
		"access_strip_path":       snap.AccessStripPath,
		"status":                  model.APIStatusPublished,
		"current_version":         version,
		"kong_service_id":         result.ServiceID,
		"kong_route_id":           result.RouteID,
	}).Error; err != nil {
		return nil, err
	}
	return s.Get(id)
}

func (s *APIService) ListVersions(apiID uint64) ([]model.APIVersion, error) {
	if _, err := s.Get(apiID); err != nil {
		return nil, err
	}
	var list []model.APIVersion
	err := s.db.Where("api_id = ?", apiID).Order("id desc").Find(&list).Error
	return list, err
}

func (s *APIService) nextVersion(apiID uint64) (string, error) {
	var count int64
	if err := s.db.Model(&model.APIVersion{}).Where("api_id = ?", apiID).Count(&count).Error; err != nil {
		return "", err
	}
	return "v" + strconv.FormatInt(count+1, 10), nil
}

func (s *APIService) withSpacePrefix(spaceID uint64, paths []string) ([]string, error) {
	var space model.Space
	if err := s.db.Select("id", "prefix").First(&space, spaceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return model.ApplyPathPrefix(space.Prefix, paths), nil
}

func (s *APIService) getGroup(id uint64) (*model.APIGroup, error) {
	var group model.APIGroup
	if err := s.db.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &group, nil
}

var accessHostPattern = regexp.MustCompile(`^(\*\.)?([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)*[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(?::\d{1,5})?$`)
var headerNamePattern = regexp.MustCompile(`^[A-Za-z0-9!#$%&'*+.^_` + "`" + `|~-]+$`)

func normalizeAccessHosts(raw string) (string, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == ';'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !accessHostPattern.MatchString(p) {
			return "", fmt.Errorf("%w: invalid access host %s", ErrBadRequest, p)
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return strings.Join(out, ","), nil
}

func normalizeAccessHeaders(in map[string][]string) (datatypes.JSON, error) {
	if len(in) == 0 {
		return datatypes.JSON([]byte("null")), nil
	}
	out := map[string][]string{}
	for name, vals := range in {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !headerNamePattern.MatchString(name) {
			return nil, fmt.Errorf("%w: invalid header name %s", ErrBadRequest, name)
		}
		cleaned := make([]string, 0, len(vals))
		seen := map[string]struct{}{}
		for _, v := range vals {
			v = strings.TrimSpace(v)
			if v == "" || strings.ContainsAny(v, "\r\n") {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			cleaned = append(cleaned, v)
		}
		if len(cleaned) == 0 {
			return nil, fmt.Errorf("%w: header %s needs a value", ErrBadRequest, name)
		}
		out[name] = cleaned
	}
	if len(out) == 0 {
		return datatypes.JSON([]byte("null")), nil
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	return datatypes.JSON(raw), nil
}

func mustHeaderJSON(headers map[string][]string) datatypes.JSON {
	raw, err := normalizeAccessHeaders(headers)
	if err != nil || len(raw) == 0 {
		return datatypes.JSON([]byte("null"))
	}
	return raw
}

func decodeHeaders(raw datatypes.JSON) map[string][]string {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var out map[string][]string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func normalizeAccessProtocols(raw string) string {
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
		return "http"
	}
	return strings.Join(out, ",")
}

func normalizeMethods(methods string) string {
	parts := strings.Split(methods, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToUpper(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ",")
}

// SpaceIDOfAPI returns space id for permission checks
func (s *APIService) SpaceIDOfAPI(apiID uint64) (uint64, error) {
	api, err := s.Get(apiID)
	if err != nil {
		return 0, err
	}
	if api.Group == nil {
		return 0, ErrNotFound
	}
	return api.Group.SpaceID, nil
}

func (s *APIService) SpaceIDOfGroup(groupID uint64) (uint64, error) {
	group, err := s.getGroup(groupID)
	if err != nil {
		return 0, err
	}
	return group.SpaceID, nil
}
