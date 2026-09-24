package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	kongclient "github.com/kelvins-io/api-gateway-manager/internal/kong"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var upstreamNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,62}$`)
var targetPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*:\d{2,5}$|^(\d{1,3}\.){3}\d{1,3}:\d{2,5}$`)

type UpstreamService struct {
	db *gorm.DB
}

func NewUpstreamService(db *gorm.DB) *UpstreamService {
	return &UpstreamService{db: db}
}

type TargetInput struct {
	Target string `json:"target"`
	Weight int    `json:"weight"`
}

type HealthInput struct {
	Active  *HealthSide `json:"active"`
	Passive *HealthSide `json:"passive"`
	Threshold *float64 `json:"threshold"`
}

type HealthSide struct {
	Type                   string `json:"type"`
	HTTPPath               string `json:"http_path"`
	Timeout                int    `json:"timeout"`
	Concurrency            int    `json:"concurrency"`
	HTTPSVerifyCertificate *bool  `json:"https_verify_certificate"`
	HealthyInterval        int    `json:"healthy_interval"`
	HealthySuccesses       int    `json:"healthy_successes"`
	HealthyHTTPStatuses    []int  `json:"healthy_http_statuses"`
	UnhealthyInterval      int    `json:"unhealthy_interval"`
	UnhealthyHTTPFailures  int    `json:"unhealthy_http_failures"`
	UnhealthyTCPFailures   int    `json:"unhealthy_tcp_failures"`
	UnhealthyTimeouts      int    `json:"unhealthy_timeouts"`
	UnhealthyHTTPStatuses  []int  `json:"unhealthy_http_statuses"`
}

type UpsertUpstreamInput struct {
	Name                   string       `json:"name" binding:"required"`
	Algorithm              string       `json:"algorithm"`
	Slots                  int          `json:"slots"`
	HashOn                 string       `json:"hash_on"`
	HashFallback           string       `json:"hash_fallback"`
	HashOnHeader           string       `json:"hash_on_header"`
	HashFallbackHeader     string       `json:"hash_fallback_header"`
	HashOnCookie           string       `json:"hash_on_cookie"`
	HashOnCookiePath       string       `json:"hash_on_cookie_path"`
	HashOnQueryArg         string       `json:"hash_on_query_arg"`
	HashFallbackQueryArg   string       `json:"hash_fallback_query_arg"`
	HashOnURICapture       string       `json:"hash_on_uri_capture"`
	HashFallbackURICapture string       `json:"hash_fallback_uri_capture"`
	Healthchecks           *HealthInput `json:"healthchecks"`
	Targets                []TargetInput `json:"targets"`
}

func (s *UpstreamService) EnsureSpace(id, spaceID uint64) error {
	var count int64
	if err := s.db.Model(&model.Upstream{}).Where("id = ? AND space_id = ?", id, spaceID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *UpstreamService) List(spaceID uint64) ([]model.Upstream, error) {
	var list []model.Upstream
	err := s.db.Preload("Targets").Where("space_id = ?", spaceID).Order("id desc").Find(&list).Error
	return list, err
}

func (s *UpstreamService) Create(spaceID uint64, in UpsertUpstreamInput) (*model.Upstream, error) {
	up, targets, err := buildUpstream(spaceID, in)
	if err != nil {
		return nil, err
	}
	var count int64
	if err := s.db.Model(&model.Upstream{}).Where("space_id = ? AND name = ?", spaceID, up.Name).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("%w: upstream name already exists", ErrConflict)
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(up).Error; err != nil {
			return err
		}
		for i := range targets {
			targets[i].UpstreamID = up.ID
		}
		if len(targets) > 0 {
			return tx.Create(&targets).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(up.ID)
}

func (s *UpstreamService) Get(id uint64) (*model.Upstream, error) {
	var up model.Upstream
	if err := s.db.Preload("Targets").First(&up, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &up, nil
}

func (s *UpstreamService) Update(ctx context.Context, id uint64, in UpsertUpstreamInput) (*model.Upstream, error) {
	current, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	up, targets, err := buildUpstream(current.SpaceID, in)
	if err != nil {
		return nil, err
	}
	var count int64
	if err := s.db.Model(&model.Upstream{}).Where("space_id = ? AND name = ? AND id <> ?", current.SpaceID, up.Name, id).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("%w: upstream name already exists", ErrConflict)
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		up.ID = current.ID
		if err := tx.Model(current).Updates(map[string]interface{}{
			"name": up.Name, "algorithm": up.Algorithm, "slots": up.Slots,
			"hash_on": up.HashOn, "hash_fallback": up.HashFallback,
			"hash_on_header": up.HashOnHeader, "hash_fallback_header": up.HashFallbackHeader,
			"hash_on_cookie": up.HashOnCookie, "hash_on_cookie_path": up.HashOnCookiePath,
			"hash_on_query_arg": up.HashOnQueryArg, "hash_fallback_query_arg": up.HashFallbackQueryArg,
			"hash_on_uri_capture": up.HashOnURICapture, "hash_fallback_uri_capture": up.HashFallbackURICapture,
			"healthchecks": up.Healthchecks,
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("upstream_id = ?", id).Delete(&model.UpstreamTarget{}).Error; err != nil {
			return err
		}
		for i := range targets {
			targets[i].UpstreamID = id
		}
		if len(targets) > 0 {
			return tx.Create(&targets).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	saved, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if err := s.resync(ctx, saved); err != nil {
		return nil, err
	}
	return saved, nil
}

func (s *UpstreamService) Delete(ctx context.Context, id uint64) error {
	up, err := s.Get(id)
	if err != nil {
		return err
	}
	var used int64
	if err := s.db.Model(&model.API{}).Where("upstream_id = ?", id).Count(&used).Error; err != nil {
		return err
	}
	if used > 0 {
		return fmt.Errorf("%w: upstream is used by apis", ErrConflict)
	}
	var bindings []model.UpstreamGateway
	if err := s.db.Where("upstream_id = ?", id).Find(&bindings).Error; err != nil {
		return err
	}
	for _, b := range bindings {
		var gw model.Gateway
		if err := s.db.First(&gw, b.GatewayID).Error; err != nil {
			continue
		}
		client, err := kongclient.New(gw.AdminAPI)
		if err != nil {
			return err
		}
		if err := client.DeleteUpstream(ctx, b.KongUpstreamID); err != nil {
			return err
		}
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("upstream_id = ?", id).Delete(&model.UpstreamGateway{}).Error; err != nil {
			return err
		}
		if err := tx.Where("upstream_id = ?", id).Delete(&model.UpstreamTarget{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Upstream{}, up.ID).Error
	})
}

func (s *UpstreamService) resync(ctx context.Context, up *model.Upstream) error {
	var bindings []model.UpstreamGateway
	if err := s.db.Where("upstream_id = ?", up.ID).Find(&bindings).Error; err != nil {
		return err
	}
	for _, b := range bindings {
		var gw model.Gateway
		if err := s.db.First(&gw, b.GatewayID).Error; err != nil {
			return err
		}
		client, err := kongclient.New(gw.AdminAPI)
		if err != nil {
			return err
		}
		kongID, err := client.SyncUpstream(ctx, upstreamToSync(*up, b.KongUpstreamID))
		if err != nil {
			return err
		}
		if kongID != b.KongUpstreamID {
			if err := s.db.Model(&b).Update("kong_upstream_id", kongID).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func buildUpstream(spaceID uint64, in UpsertUpstreamInput) (*model.Upstream, []model.UpstreamTarget, error) {
	name := strings.TrimSpace(in.Name)
	if !upstreamNamePattern.MatchString(name) {
		return nil, nil, fmt.Errorf("%w: invalid upstream name", ErrBadRequest)
	}
	algo := in.Algorithm
	if algo == "" {
		algo = "round-robin"
	}
	switch algo {
	case "round-robin", "consistent-hashing", "least-connections", "latency":
	default:
		return nil, nil, fmt.Errorf("%w: unsupported algorithm", ErrBadRequest)
	}
	slots := in.Slots
	if slots == 0 {
		slots = 10000
	}
	if slots < 10 || slots > 65536 {
		return nil, nil, fmt.Errorf("%w: slots must be between 10 and 65536", ErrBadRequest)
	}
	raw, err := encodeHealth(in.Healthchecks)
	if err != nil {
		return nil, nil, err
	}
	targets := make([]model.UpstreamTarget, 0, len(in.Targets))
	seen := map[string]struct{}{}
	for _, t := range in.Targets {
		target := strings.TrimSpace(t.Target)
		if target == "" {
			continue
		}
		if !targetPattern.MatchString(target) {
			return nil, nil, fmt.Errorf("%w: target must look like host:port", ErrBadRequest)
		}
		if _, ok := seen[target]; ok {
			continue
		}
		seen[target] = struct{}{}
		weight := t.Weight
		if weight == 0 {
			weight = 100
		}
		if weight < 0 || weight > 65535 {
			return nil, nil, fmt.Errorf("%w: target weight out of range", ErrBadRequest)
		}
		targets = append(targets, model.UpstreamTarget{Target: target, Weight: weight})
	}
	up := &model.Upstream{
		SpaceID: spaceID, Name: name, Algorithm: algo, Slots: slots,
		HashOn: in.HashOn, HashFallback: in.HashFallback,
		HashOnHeader: in.HashOnHeader, HashFallbackHeader: in.HashFallbackHeader,
		HashOnCookie: in.HashOnCookie, HashOnCookiePath: in.HashOnCookiePath,
		HashOnQueryArg: in.HashOnQueryArg, HashFallbackQueryArg: in.HashFallbackQueryArg,
		HashOnURICapture: in.HashOnURICapture, HashFallbackURICapture: in.HashFallbackURICapture,
		Healthchecks: datatypes.JSON(raw),
	}
	return up, targets, nil
}

func encodeHealth(in *HealthInput) ([]byte, error) {
	if in == nil {
		return []byte("null"), nil
	}
	return json.Marshal(in)
}

func healthToKong(raw datatypes.JSON) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var in HealthInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return json.RawMessage(raw)
	}
	body := map[string]interface{}{}
	if in.Threshold != nil {
		body["threshold"] = *in.Threshold
	}
	if in.Active != nil {
		body["active"] = sideToKong(in.Active, true)
	}
	if in.Passive != nil {
		body["passive"] = sideToKong(in.Passive, false)
	}
	out, err := json.Marshal(body)
	if err != nil {
		return nil
	}
	return out
}

func sideToKong(in *HealthSide, active bool) map[string]interface{} {
	typ := in.Type
	if typ == "" {
		typ = "http"
	}
	healthy := map[string]interface{}{
		"successes":     in.HealthySuccesses,
		"http_statuses": statusesOr(in.HealthyHTTPStatuses, healthyStatuses(active)),
	}
	unhealthy := map[string]interface{}{
		"http_failures": in.UnhealthyHTTPFailures,
		"tcp_failures":  in.UnhealthyTCPFailures,
		"timeouts":      in.UnhealthyTimeouts,
		"http_statuses": statusesOr(in.UnhealthyHTTPStatuses, unhealthyStatuses(active)),
	}
	if active {
		healthy["interval"] = in.HealthyInterval
		unhealthy["interval"] = in.UnhealthyInterval
	}
	out := map[string]interface{}{
		"type":      typ,
		"healthy":   healthy,
		"unhealthy": unhealthy,
	}
	if active {
		path := in.HTTPPath
		if path == "" {
			path = "/"
		}
		out["http_path"] = path
		timeout := in.Timeout
		if timeout == 0 {
			timeout = 1
		}
		out["timeout"] = timeout
		concurrency := in.Concurrency
		if concurrency == 0 {
			concurrency = 10
		}
		out["concurrency"] = concurrency
		verify := true
		if in.HTTPSVerifyCertificate != nil {
			verify = *in.HTTPSVerifyCertificate
		}
		out["https_verify_certificate"] = verify
	}
	return out
}

func statusesOr(got, fallback []int) []int {
	if len(got) == 0 {
		return fallback
	}
	return got
}

func healthyStatuses(active bool) []int {
	if active {
		return []int{200, 302}
	}
	return []int{200, 201, 202, 203, 204, 205, 206, 207, 208, 226, 300, 301, 302, 303, 304, 305, 306, 307, 308}
}

func unhealthyStatuses(active bool) []int {
	if active {
		return []int{429, 404, 500, 501, 502, 503, 504, 505}
	}
	return []int{429, 500, 503}
}

func upstreamToSync(up model.Upstream, existingID string) kongclient.UpstreamSync {
	targets := make([]kongclient.TargetSync, 0, len(up.Targets))
	for _, t := range up.Targets {
		targets = append(targets, kongclient.TargetSync{Target: t.Target, Weight: t.Weight})
	}
	return kongclient.UpstreamSync{
		KongName: model.KongUpstreamName(up.SpaceID, up.Name), ExistingID: existingID,
		Algorithm: up.Algorithm, Slots: up.Slots,
		HashOn: up.HashOn, HashFallback: up.HashFallback,
		HashOnHeader: up.HashOnHeader, HashFallbackHeader: up.HashFallbackHeader,
		HashOnCookie: up.HashOnCookie, HashOnCookiePath: up.HashOnCookiePath,
		HashOnQueryArg: up.HashOnQueryArg, HashFallbackQueryArg: up.HashFallbackQueryArg,
		HashOnURICapture: up.HashOnURICapture, HashFallbackURICapture: up.HashFallbackURICapture,
		Healthchecks: healthToKong(up.Healthchecks), Targets: targets,
	}
}
