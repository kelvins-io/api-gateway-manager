package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Name            string  `json:"name" binding:"required,min=1,max=128"`
	Path            string  `json:"path" binding:"required,min=1,max=2048"`
	Methods         string  `json:"methods" binding:"required"`
	AccessProtocols string  `json:"access_protocols"`
	Protocol        string  `json:"protocol"`
	HostKind        string  `json:"host_kind"`
	Host            string  `json:"host"`
	UpstreamID      *uint64 `json:"upstream_id"`
	Port            int     `json:"port"`
	ServicePath     string  `json:"service_path"`
	Retries         *int    `json:"retries"`
	ConnectTimeout  *int    `json:"connect_timeout"`
	WriteTimeout    *int    `json:"write_timeout"`
	ReadTimeout     *int    `json:"read_timeout"`
	StripPath       *bool   `json:"strip_path"`
}

type UpdateAPIInput struct {
	Name            string  `json:"name" binding:"omitempty,min=1,max=128"`
	Path            string  `json:"path" binding:"omitempty,min=1,max=2048"`
	Methods         string  `json:"methods"`
	AccessProtocols string  `json:"access_protocols"`
	Protocol        string  `json:"protocol"`
	HostKind        string  `json:"host_kind"`
	Host            string  `json:"host"`
	UpstreamID      *uint64 `json:"upstream_id"`
	Port            *int    `json:"port"`
	ServicePath     string  `json:"service_path"`
	Retries         *int    `json:"retries"`
	ConnectTimeout  *int    `json:"connect_timeout"`
	WriteTimeout    *int    `json:"write_timeout"`
	ReadTimeout     *int    `json:"read_timeout"`
	StripPath       *bool   `json:"strip_path"`
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
	if in.StripPath != nil {
		strip = *in.StripPath
	}
	paths := model.SplitPaths(in.Path)
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: at least one path is required", ErrBadRequest)
	}
	paths, err = s.withSpacePrefix(group.SpaceID, paths)
	if err != nil {
		return nil, err
	}
	svcFields, err := s.normalizeService(group.SpaceID, serviceInput{
		Protocol: in.Protocol, HostKind: in.HostKind, Host: in.Host, UpstreamID: in.UpstreamID,
		Port: in.Port, ServicePath: in.ServicePath, Retries: in.Retries,
		ConnectTimeout: in.ConnectTimeout, WriteTimeout: in.WriteTimeout, ReadTimeout: in.ReadTimeout,
	}, true)
	if err != nil {
		return nil, err
	}
	api := &model.API{
		GroupID:         groupID,
		Name:            in.Name,
		Path:            strings.Join(paths, ","),
		Methods:         normalizeMethods(in.Methods),
		AccessProtocols: normalizeAccessProtocols(in.AccessProtocols),
		Protocol:        svcFields.Protocol,
		HostKind:        svcFields.HostKind,
		Host:            svcFields.Host,
		UpstreamID:      svcFields.UpstreamID,
		Port:            svcFields.Port,
		ServicePath:     svcFields.ServicePath,
		Retries:         svcFields.Retries,
		ConnectTimeout:  svcFields.ConnectTimeout,
		WriteTimeout:    svcFields.WriteTimeout,
		ReadTimeout:     svcFields.ReadTimeout,
		StripPath:       strip,
		Status:          model.APIStatusDraft,
	}
	if err := s.db.Create(api).Error; err != nil {
		return nil, err
	}
	return s.Get(api.ID)
}

func (s *APIService) ListByGroup(groupID uint64) ([]model.API, error) {
	var list []model.API
	err := s.db.Where("group_id = ?", groupID).Order("id desc").Find(&list).Error
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
	if in.Path != "" {
		paths := model.SplitPaths(in.Path)
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
		updates["path"] = strings.Join(paths, ",")
	}
	if in.Methods != "" {
		updates["methods"] = normalizeMethods(in.Methods)
	}
	if in.AccessProtocols != "" {
		updates["access_protocols"] = normalizeAccessProtocols(in.AccessProtocols)
	}
	if in.Protocol != "" || in.HostKind != "" || in.Host != "" || in.UpstreamID != nil || in.Port != nil || in.ServicePath != "" || in.Retries != nil || in.ConnectTimeout != nil || in.WriteTimeout != nil || in.ReadTimeout != nil {
		port := api.Port
		if in.Port != nil {
			port = *in.Port
		}
		svcFields, err := s.normalizeService(api.Group.SpaceID, serviceInput{
			Protocol: in.Protocol, HostKind: in.HostKind, Host: in.Host, UpstreamID: in.UpstreamID,
			Port: port, ServicePath: in.ServicePath, Retries: in.Retries,
			ConnectTimeout: in.ConnectTimeout, WriteTimeout: in.WriteTimeout, ReadTimeout: in.ReadTimeout,
		}, false)
		if err != nil {
			return nil, err
		}
		if svcFields.Protocol == "" {
			svcFields.Protocol = api.Protocol
		}
		if svcFields.HostKind == "" {
			svcFields.HostKind = api.HostKind
		}
		if svcFields.ServicePath == "" {
			svcFields.ServicePath = api.ServicePath
		}
		updates["protocol"] = svcFields.Protocol
		updates["host_kind"] = svcFields.HostKind
		updates["host"] = svcFields.Host
		if svcFields.UpstreamID == nil {
			updates["upstream_id"] = gorm.Expr("NULL")
		} else {
			updates["upstream_id"] = *svcFields.UpstreamID
		}
		updates["port"] = svcFields.Port
		updates["service_path"] = svcFields.ServicePath
		updates["retries"] = svcFields.Retries
		updates["connect_timeout"] = svcFields.ConnectTimeout
		updates["write_timeout"] = svcFields.WriteTimeout
		updates["read_timeout"] = svcFields.ReadTimeout
	}
	if in.StripPath != nil {
		updates["strip_path"] = *in.StripPath
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

	if snap.HostKind == model.HostKindUpstream && snap.UpstreamID > 0 {
		kongHost, syncErr := s.syncUpstreamGateway(ctx, snap.UpstreamID, api.Group.Gateway)
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
		"name":             snap.Name,
		"path":             snap.Path,
		"methods":          snap.Methods,
		"access_protocols": snap.AccessProtocols,
		"upstream_url":     snap.UpstreamURL,
		"protocol":         snap.Protocol,
		"host_kind":        snap.HostKind,
		"host":             snap.Host,
		"upstream_id":      nilIfZero(snap.UpstreamID),
		"port":             snap.Port,
		"service_path":     snap.ServicePath,
		"retries":          snap.Retries,
		"connect_timeout":  snap.ConnectTimeout,
		"write_timeout":    snap.WriteTimeout,
		"read_timeout":     snap.ReadTimeout,
		"strip_path":       snap.StripPath,
		"status":           model.APIStatusPublished,
		"current_version":  version,
		"kong_service_id":  result.ServiceID,
		"kong_route_id":    result.RouteID,
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
