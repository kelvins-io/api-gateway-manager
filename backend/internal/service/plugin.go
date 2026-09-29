package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	kongclient "github.com/kelvins-io/api-gateway-manager/internal/kong"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var pluginNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,62}$`)

// allowedPlugins is Kong Gateway 3.4.2 OSS bundled plugins plus custom plugins
// (e.g. proxy-cache-advanced, response-ratelimiting-advanced, grpc-gateway-advanced,
// grpc-web-advanced, request-gzip, response-gzip).
var allowedPlugins = map[string]struct{}{
	"acl":                            {},
	"acme":                           {},
	"aws-lambda":                     {},
	"azure-functions":                {},
	"basic-auth":                     {},
	"bot-detection":                  {},
	"correlation-id":                 {},
	"cors":                           {},
	"datadog":                        {},
	"file-log":                       {},
	"grpc-gateway":                   {},
	"grpc-gateway-advanced":          {},
	"grpc-web":                       {},
	"grpc-web-advanced":              {},
	"hmac-auth":                      {},
	"http-log":                       {},
	"ip-restriction":                 {},
	"jwt":                            {},
	"key-auth":                       {},
	"ldap-auth":                      {},
	"loggly":                         {},
	"oauth2":                         {},
	"opentelemetry":                  {},
	"post-function":                  {},
	"pre-function":                   {},
	"prometheus":                     {},
	"proxy-cache":                    {},
	"proxy-cache-advanced":           {},
	"rate-limiting":                  {},
	"request-gzip":                   {},
	"request-size-limiting":          {},
	"request-termination":            {},
	"request-transformer":            {},
	"response-gzip":                  {},
	"response-ratelimiting":          {},
	"response-ratelimiting-advanced": {},
	"response-transformer":           {},
	"session":                        {},
	"statsd":                         {},
	"syslog":                         {},
	"tcp-log":                        {},
	"udp-log":                        {},
	"zipkin":                         {},
}

type PluginService struct {
	db *gorm.DB
}

func NewPluginService(db *gorm.DB) *PluginService {
	return &PluginService{db: db}
}

type UpsertPluginInput struct {
	Name    string                 `json:"name" binding:"required"`
	Plugin  string                 `json:"plugin" binding:"required"`
	Config  map[string]interface{} `json:"config"`
	Enabled *bool                  `json:"enabled"`
}

func (s *PluginService) EnsureSpace(id uint64, spaceID uint64) error {
	var count int64
	if err := s.db.Model(&model.Plugin{}).Where("id = ? AND space_id = ?", id, spaceID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PluginService) List(spaceID uint64) ([]model.Plugin, error) {
	var list []model.Plugin
	err := s.db.Where("space_id = ?", spaceID).Order("id desc").Find(&list).Error
	return list, err
}

func (s *PluginService) ListAPIs(id uint64, spaceID uint64) ([]model.API, error) {
	if err := s.EnsureSpace(id, spaceID); err != nil {
		return nil, err
	}
	var list []model.API
	err := s.db.Joins("JOIN api_plugins ON api_plugins.api_id = apis.id").
		Joins("JOIN api_groups ON api_groups.id = apis.group_id").
		Where("api_plugins.plugin_id = ? AND api_groups.space_id = ?", id, spaceID).
		Preload("Group").
		Order("apis.id desc").
		Find(&list).Error
	return list, err
}

func (s *PluginService) Create(_ context.Context, spaceID uint64, in UpsertPluginInput) (*model.Plugin, error) {
	item, err := buildPlugin(spaceID, in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureUnique(spaceID, item.Name, 0); err != nil {
		return nil, err
	}
	if err := s.db.Select("SpaceID", "Name", "Plugin", "Config", "Enabled").Create(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

func (s *PluginService) Update(ctx context.Context, id uint64, in UpsertPluginInput) (*model.Plugin, error) {
	current, err := s.get(id)
	if err != nil {
		return nil, err
	}
	item, err := buildPlugin(current.SpaceID, in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureUnique(current.SpaceID, item.Name, id); err != nil {
		return nil, err
	}
	if item.Plugin != current.Plugin {
		return nil, fmt.Errorf("%w: plugin type cannot be changed", ErrBadRequest)
	}
	if err := s.db.Model(current).Updates(map[string]interface{}{
		"name":    item.Name,
		"config":  item.Config,
		"enabled": item.Enabled,
	}).Error; err != nil {
		return nil, err
	}
	saved, err := s.get(id)
	if err != nil {
		return nil, err
	}
	if err := syncPluginOnPublishedAPIs(ctx, s.db, id); err != nil {
		return nil, err
	}
	return saved, nil
}

func (s *PluginService) Delete(ctx context.Context, id uint64) error {
	if _, err := s.get(id); err != nil {
		return err
	}
	var apis []model.API
	if err := s.db.Joins("JOIN api_plugins ON api_plugins.api_id = apis.id").
		Where("api_plugins.plugin_id = ? AND apis.status = ?", id, model.APIStatusPublished).
		Preload("Group.Gateway").Preload("Plugins").Preload("Consumers.Credentials").Find(&apis).Error; err != nil {
		return err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM api_plugins WHERE plugin_id = ?", id).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Plugin{}, id).Error
	}); err != nil {
		return err
	}
	for i := range apis {
		kept := make([]model.Plugin, 0, len(apis[i].Plugins))
		for _, p := range apis[i].Plugins {
			if p.ID != id {
				kept = append(kept, p)
			}
		}
		apis[i].Plugins = kept
		if err := applyAPIPlugins(ctx, &apis[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *PluginService) get(id uint64) (*model.Plugin, error) {
	var item model.Plugin
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (s *PluginService) ensureUnique(spaceID uint64, name string, excludeID uint64) error {
	var count int64
	q := s.db.Model(&model.Plugin{}).Where("space_id = ? AND name = ?", spaceID, name)
	if excludeID > 0 {
		q = q.Where("id <> ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: plugin name already exists", ErrConflict)
	}
	return nil
}

func buildPlugin(spaceID uint64, in UpsertPluginInput) (*model.Plugin, error) {
	name := strings.TrimSpace(in.Name)
	if !pluginNamePattern.MatchString(name) {
		return nil, fmt.Errorf("%w: invalid plugin name", ErrBadRequest)
	}
	kind := strings.TrimSpace(strings.ToLower(in.Plugin))
	if _, ok := allowedPlugins[kind]; !ok {
		return nil, fmt.Errorf("%w: unsupported plugin", ErrBadRequest)
	}
	cfg, err := normalizePluginConfig(kind, in.Config)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	enabled := true
	if in.Enabled != nil {
		enabled = *in.Enabled
	}
	return &model.Plugin{
		SpaceID: spaceID,
		Name:    name,
		Plugin:  kind,
		Config:  datatypes.JSON(raw),
		Enabled: enabled,
	}, nil
}

func normalizePluginConfig(kind string, in map[string]interface{}) (map[string]interface{}, error) {
	switch kind {
	case "rate-limiting":
		limits := map[string]int{}
		for _, key := range []string{"second", "minute", "hour", "day", "month", "year"} {
			n := intField(in, key)
			if n < 0 {
				return nil, fmt.Errorf("%w: %s must be >= 0", ErrBadRequest, key)
			}
			if n > 0 {
				limits[key] = n
			}
		}
		if len(limits) == 0 {
			return nil, fmt.Errorf("%w: rate-limiting needs at least one limit", ErrBadRequest)
		}
		limitBy := strField(in, "limit_by", "consumer")
		if !oneOf(limitBy, "consumer", "credential", "ip", "service", "header", "path") {
			return nil, fmt.Errorf("%w: invalid limit_by", ErrBadRequest)
		}
		policy := strField(in, "policy", "local")
		if !oneOf(policy, "local", "cluster", "redis") {
			return nil, fmt.Errorf("%w: invalid policy", ErrBadRequest)
		}
		out := map[string]interface{}{
			"limit_by":            limitBy,
			"policy":              policy,
			"fault_tolerant":      boolField(in, "fault_tolerant", true),
			"hide_client_headers": boolField(in, "hide_client_headers", false),
		}
		for k, v := range limits {
			out[k] = v
		}
		return out, nil
	case "cors":
		origins := csvField(in, "origins")
		if len(origins) == 0 {
			origins = []string{"*"}
		}
		methods := csvField(in, "methods")
		if len(methods) == 0 {
			methods = []string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE", "OPTIONS", "TRACE", "CONNECT"}
		}
		headers := csvField(in, "headers")
		if len(headers) == 0 {
			headers = []string{"*"}
		}
		return map[string]interface{}{
			"origins":            origins,
			"methods":            methods,
			"headers":            headers,
			"exposed_headers":    listOrEmpty(csvField(in, "exposed_headers")),
			"credentials":        boolField(in, "credentials", false),
			"max_age":            intField(in, "max_age"),
			"preflight_continue": boolField(in, "preflight_continue", false),
		}, nil
	case "key-auth":
		names := csvField(in, "key_names")
		if len(names) == 0 {
			names = []string{"apikey"}
		}
		return map[string]interface{}{
			"key_names":        names,
			"hide_credentials": boolField(in, "hide_credentials", false),
			"key_in_header":    boolField(in, "key_in_header", true),
			"key_in_query":     boolField(in, "key_in_query", true),
			"key_in_body":      boolField(in, "key_in_body", false),
		}, nil
	case "acl":
		allow := csvField(in, "allow")
		deny := csvField(in, "deny")
		if len(allow) == 0 && len(deny) == 0 {
			return nil, fmt.Errorf("%w: acl requires allow or deny", ErrBadRequest)
		}
		if len(allow) > 0 && len(deny) > 0 {
			return nil, fmt.Errorf("%w: acl allow and deny are mutually exclusive", ErrBadRequest)
		}
		out := map[string]interface{}{"hide_groups_header": boolField(in, "hide_groups_header", false)}
		if len(allow) > 0 {
			out["allow"] = allow
		}
		if len(deny) > 0 {
			out["deny"] = deny
		}
		return out, nil
	case "ip-restriction":
		allow := csvField(in, "allow")
		deny := csvField(in, "deny")
		if len(allow) == 0 && len(deny) == 0 {
			return nil, fmt.Errorf("%w: ip-restriction requires allow or deny", ErrBadRequest)
		}
		if len(allow) > 0 && len(deny) > 0 {
			return nil, fmt.Errorf("%w: ip-restriction allow and deny are mutually exclusive", ErrBadRequest)
		}
		out := map[string]interface{}{}
		if len(allow) > 0 {
			out["allow"] = allow
		}
		if len(deny) > 0 {
			out["deny"] = deny
		}
		return out, nil
	case "request-size-limiting":
		size := intField(in, "allowed_payload_size")
		if size <= 0 {
			return nil, fmt.Errorf("%w: allowed_payload_size must be > 0", ErrBadRequest)
		}
		unit := strField(in, "size_unit", "megabytes")
		if !oneOf(unit, "megabytes", "kilobytes", "bytes") {
			return nil, fmt.Errorf("%w: invalid size_unit", ErrBadRequest)
		}
		return map[string]interface{}{"allowed_payload_size": size, "size_unit": unit}, nil
	case "jwt":
		headers := csvField(in, "header_names")
		if len(headers) == 0 {
			headers = []string{"authorization"}
		}
		return map[string]interface{}{
			"uri_param_names":  listOrEmpty(csvField(in, "uri_param_names")),
			"header_names":     headers,
			"cookie_names":     listOrEmpty(csvField(in, "cookie_names")),
			"claims_to_verify": listOrEmpty(csvField(in, "claims_to_verify")),
		}, nil
	case "basic-auth":
		return map[string]interface{}{"hide_credentials": boolField(in, "hide_credentials", false)}, nil
	case "hmac-auth":
		out := map[string]interface{}{"hide_credentials": boolField(in, "hide_credentials", false)}
		if skew := intField(in, "clock_skew"); skew > 0 {
			out["clock_skew"] = skew
		}
		return out, nil
	case "request-termination":
		code := intField(in, "status_code")
		if code == 0 {
			code = 503
		}
		if code < 100 || code > 599 {
			return nil, fmt.Errorf("%w: status_code must be between 100 and 599", ErrBadRequest)
		}
		out := map[string]interface{}{"status_code": code}
		if msg := strings.TrimSpace(strField(in, "message", "")); msg != "" {
			out["message"] = msg
		}
		if ct := strings.TrimSpace(strField(in, "content_type", "")); ct != "" {
			out["content_type"] = ct
		}
		return out, nil
	case "correlation-id":
		header := strField(in, "header_name", "Kong-Request-ID")
		if header == "" {
			return nil, fmt.Errorf("%w: header_name is required", ErrBadRequest)
		}
		gen := strField(in, "generator", "uuid")
		if !oneOf(gen, "uuid", "uuid#counter", "tracker") {
			return nil, fmt.Errorf("%w: invalid generator", ErrBadRequest)
		}
		return map[string]interface{}{
			"header_name":     header,
			"generator":       gen,
			"echo_downstream": boolField(in, "echo_downstream", false),
		}, nil
	default:
		// Pass through raw config for other Kong 3.4.2 OSS plugins; Kong validates on publish.
		if in == nil {
			return map[string]interface{}{}, nil
		}
		out := make(map[string]interface{}, len(in))
		for k, v := range in {
			out[k] = v
		}
		return out, nil
	}
}

func pluginsToSync(api *model.API) ([]kongclient.PluginSync, error) {
	out := make([]kongclient.PluginSync, 0, len(api.Plugins))
	for _, p := range api.Plugins {
		if !p.Enabled {
			continue
		}
		var cfg map[string]interface{}
		if len(p.Config) > 0 {
			if err := json.Unmarshal(p.Config, &cfg); err != nil {
				return nil, err
			}
		}
		if cfg == nil {
			cfg = map[string]interface{}{}
		}
		out = append(out, kongclient.PluginSync{
			Name:         p.Plugin,
			InstanceName: fmt.Sprintf("agm-s%d-api%d-p%d", p.SpaceID, api.ID, p.ID),
			Config:       cfg,
			Enabled:      true,
			Tag:          "agm-plugin-" + strconv.FormatUint(p.ID, 10),
		})
	}
	return withAPIAuth(api, out)
}

func withAPIAuth(api *model.API, specs []kongclient.PluginSync) ([]kongclient.PluginSync, error) {
	if !api.AuthEnabled || api.AuthPlugin == "" {
		return specs, nil
	}
	var cfg map[string]interface{}
	if len(api.AuthConfig) > 0 && string(api.AuthConfig) != "null" {
		if err := json.Unmarshal(api.AuthConfig, &cfg); err != nil {
			return nil, err
		}
	}
	if cfg == nil {
		cfg = map[string]interface{}{}
	}
	group := model.APIACLGroup(api.ID)
	filtered := make([]kongclient.PluginSync, 0, len(specs)+2)
	for _, spec := range specs {
		if spec.Name == api.AuthPlugin || spec.Name == "acl" {
			continue
		}
		filtered = append(filtered, spec)
	}
	filtered = append(filtered, kongclient.PluginSync{
		Name:         api.AuthPlugin,
		InstanceName: fmt.Sprintf("agm-api%d-%s", api.ID, api.AuthPlugin),
		Config:       cfg,
		Enabled:      true,
		Tag:          "agm-api-auth",
	}, kongclient.PluginSync{
		Name:         "acl",
		InstanceName: fmt.Sprintf("agm-api%d-acl", api.ID),
		Config:       map[string]interface{}{"allow": []string{group}, "hide_groups_header": false},
		Enabled:      true,
		Tag:          "agm-api-auth",
	})
	return filtered, nil
}

func managedPluginTag(apiID uint64) string {
	return "agm-api-" + strconv.FormatUint(apiID, 10)
}

func applyAPIPlugins(ctx context.Context, api *model.API) error {
	if api.Status != model.APIStatusPublished || api.KongServiceID == "" {
		return nil
	}
	if api.Group == nil || api.Group.Gateway == nil {
		return fmt.Errorf("%w: gateway not configured", ErrBadRequest)
	}
	specs, err := pluginsToSync(api)
	if err != nil {
		return err
	}
	client, err := kongclient.New(api.Group.Gateway.AdminAPI)
	if err != nil {
		return err
	}
	if err := client.ReplaceServicePlugins(ctx, api.KongServiceID, managedPluginTag(api.ID), specs); err != nil {
		return fmt.Errorf("sync plugins: %w", err)
	}
	return nil
}

func syncPluginOnPublishedAPIs(ctx context.Context, db *gorm.DB, pluginID uint64) error {
	var apis []model.API
	if err := db.Joins("JOIN api_plugins ON api_plugins.api_id = apis.id").
		Where("api_plugins.plugin_id = ? AND apis.status = ?", pluginID, model.APIStatusPublished).
		Preload("Group.Gateway").Preload("Plugins").Preload("Consumers.Credentials").Find(&apis).Error; err != nil {
		return err
	}
	for i := range apis {
		if err := applyAPIPlugins(ctx, &apis[i]); err != nil {
			return err
		}
	}
	return nil
}

func strField(in map[string]interface{}, key, fallback string) string {
	if in == nil {
		return fallback
	}
	v, ok := in[key]
	if !ok || v == nil {
		return fallback
	}
	switch t := v.(type) {
	case string:
		t = strings.TrimSpace(t)
		if t == "" {
			return fallback
		}
		return t
	default:
		s := strings.TrimSpace(fmt.Sprint(t))
		if s == "" {
			return fallback
		}
		return s
	}
}

func intField(in map[string]interface{}, key string) int {
	if in == nil {
		return 0
	}
	v, ok := in[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	default:
		n, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(t)))
		return n
	}
}

func boolField(in map[string]interface{}, key string, fallback bool) bool {
	if in == nil {
		return fallback
	}
	v, ok := in[key]
	if !ok || v == nil {
		return fallback
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, err := strconv.ParseBool(strings.TrimSpace(t))
		if err != nil {
			return fallback
		}
		return b
	default:
		return fallback
	}
}

func csvField(in map[string]interface{}, key string) []string {
	if in == nil {
		return nil
	}
	v, ok := in[key]
	if !ok || v == nil {
		return nil
	}
	var parts []string
	switch t := v.(type) {
	case string:
		parts = strings.FieldsFunc(t, func(r rune) bool {
			return r == ',' || r == '\n' || r == ';'
		})
	case []string:
		parts = t
	case []interface{}:
		for _, item := range t {
			parts = append(parts, fmt.Sprint(item))
		}
	default:
		parts = []string{fmt.Sprint(t)}
	}
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func listOrEmpty(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}

func oneOf(v string, options ...string) bool {
	for _, o := range options {
		if v == o {
			return true
		}
	}
	return false
}
