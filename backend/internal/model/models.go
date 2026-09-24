package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/datatypes"
)

const (
	RoleSystemAdmin = "system_admin"
	RoleSpaceAdmin  = "space_admin"
	RoleMember      = "member"

	APIStatusDraft     = "draft"
	APIStatusPublished = "published"
	APIStatusOffline   = "offline"

	HostKindDirect   = "direct"
	HostKindUpstream = "upstream"
)

type User struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"size:64;uniqueIndex;not null" json:"username"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	Role         string    `gorm:"size:32;not null;default:member" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Space struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;uniqueIndex;not null" json:"name"`
	Description string    `gorm:"size:512" json:"description"`
	Prefix      string    `gorm:"size:128;not null;default:''" json:"prefix"` // path prefix, e.g. /order
	OwnerID     uint64    `gorm:"not null;index" json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	GroupCount  int64     `gorm:"-" json:"group_count"`
}

type SpaceMember struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	SpaceID   uint64    `gorm:"not null;uniqueIndex:idx_space_user" json:"space_id"`
	UserID    uint64    `gorm:"not null;uniqueIndex:idx_space_user" json:"user_id"`
	Role      string    `gorm:"size:32;not null;default:member" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	User  *User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Space *Space `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
}

type Gateway struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:128;uniqueIndex;not null" json:"name"`
	AdminAPI    string    `gorm:"size:512;not null" json:"admin_api"`
	Domain      string    `gorm:"size:255;not null;default:''" json:"domain"`
	NetworkZone string    `gorm:"size:128;not null" json:"network_zone"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type APIGroup struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	SpaceID   uint64    `gorm:"not null;index" json:"space_id"`
	GatewayID uint64    `gorm:"not null;index" json:"gateway_id"`
	Name      string    `gorm:"size:128;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Gateway  *Gateway `gorm:"foreignKey:GatewayID" json:"gateway,omitempty"`
	Space    *Space   `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
	APICount int64    `gorm:"-" json:"api_count"`
}

type API struct {
	ID                    uint64         `gorm:"primaryKey" json:"id"`
	GroupID               uint64         `gorm:"not null;index" json:"group_id"`
	Name                  string         `gorm:"size:128;not null" json:"name"`
	AccessPath            string         `gorm:"size:2048;not null" json:"access_path"`
	AccessMethods         string         `gorm:"size:128;not null" json:"access_methods"`
	AccessProtocols       string         `gorm:"size:64;not null;default:http" json:"access_protocols"`
	AccessHosts           string         `gorm:"size:2048" json:"access_hosts"`
	AccessHeaders         datatypes.JSON `gorm:"type:jsonb" json:"access_headers"`
	UpstreamURL           string         `gorm:"size:512" json:"upstream_url"`
	ServiceProtocol       string         `gorm:"size:16;not null;default:http" json:"service_protocol"`
	ServiceHostKind       string         `gorm:"size:16;not null;default:direct" json:"service_host_kind"` // direct | upstream
	ServiceHost           string         `gorm:"size:255" json:"service_host"`
	ServiceUpstreamID     *uint64        `gorm:"index" json:"service_upstream_id,omitempty"`
	ServicePort           int            `gorm:"not null;default:80" json:"service_port"`
	ServicePath           string         `gorm:"size:512;not null;default:/" json:"service_path"`
	ServiceRetries        int            `gorm:"not null;default:5" json:"service_retries"`
	ServiceConnectTimeout int            `gorm:"not null;default:60000" json:"service_connect_timeout"`
	ServiceWriteTimeout   int            `gorm:"not null;default:60000" json:"service_write_timeout"`
	ServiceReadTimeout    int            `gorm:"not null;default:60000" json:"service_read_timeout"`
	AccessStripPath       bool           `gorm:"default:true" json:"access_strip_path"`
	AuthEnabled           bool           `gorm:"not null;default:false" json:"auth_enabled"`
	AuthPlugin            string         `gorm:"size:32;not null;default:''" json:"auth_plugin"`
	AuthConfig            datatypes.JSON `gorm:"type:jsonb" json:"auth_config"`
	Status                string         `gorm:"size:32;not null;default:draft" json:"status"`
	CurrentVersion        string         `gorm:"size:32" json:"current_version"`
	KongServiceID         string         `gorm:"size:64" json:"kong_service_id"`
	KongRouteID           string         `gorm:"size:64" json:"kong_route_id"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`

	Group     *APIGroup  `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	Plugins   []Plugin   `gorm:"many2many:api_plugins;" json:"plugins,omitempty"`
	Consumers []Consumer `gorm:"many2many:api_consumers;" json:"consumers,omitempty"`
}

func APIACLGroup(id uint64) string {
	return fmt.Sprintf("G-%d", id)
}

// Plugin is a space-scoped Kong plugin template. APIs can attach many plugins.
type Plugin struct {
	ID        uint64         `gorm:"primaryKey" json:"id"`
	SpaceID   uint64         `gorm:"not null;uniqueIndex:idx_space_plugin_name" json:"space_id"`
	Name      string         `gorm:"size:128;not null;uniqueIndex:idx_space_plugin_name" json:"name"`
	Plugin    string         `gorm:"size:64;not null" json:"plugin"`
	Config    datatypes.JSON `gorm:"type:jsonb;not null" json:"config"`
	Enabled   bool           `gorm:"not null;default:true" json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type APIVersion struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	APIID          uint64         `gorm:"not null;index;uniqueIndex:idx_api_version" json:"api_id"`
	Version        string         `gorm:"size:32;not null;uniqueIndex:idx_api_version" json:"version"`
	ConfigSnapshot datatypes.JSON `gorm:"type:jsonb;not null" json:"config_snapshot"`
	PublishedAt    time.Time      `json:"published_at"`
	CreatedAt      time.Time      `json:"created_at"`
}

// APIConfigSnapshot is stored in api_versions.config_snapshot
type APIConfigSnapshot struct {
	Name                  string              `json:"name"`
	AccessPath            string              `json:"access_path"`
	AccessMethods         string              `json:"access_methods"`
	LegacyPath            string              `json:"path,omitempty"`
	LegacyMethods         string              `json:"methods,omitempty"`
	AccessProtocols       string              `json:"access_protocols"`
	AccessHosts           string              `json:"access_hosts,omitempty"`
	AccessHeaders         map[string][]string `json:"access_headers,omitempty"`
	UpstreamURL           string              `json:"upstream_url,omitempty"`
	ServiceProtocol       string              `json:"service_protocol"`
	ServiceHostKind       string              `json:"service_host_kind"`
	ServiceHost           string              `json:"service_host"`
	ServiceUpstreamID     uint64              `json:"service_upstream_id,omitempty"`
	LegacyProtocol        string              `json:"protocol,omitempty"`
	LegacyHostKind        string              `json:"host_kind,omitempty"`
	LegacyHost            string              `json:"host,omitempty"`
	LegacyUpstreamID      *uint64             `json:"upstream_id,omitempty"`
	KongHost              string              `json:"kong_host"`
	ServicePort           int                 `json:"service_port"`
	LegacyPort            *int                `json:"port,omitempty"`
	ServicePath           string              `json:"service_path"`
	ServiceRetries        int                 `json:"service_retries"`
	ServiceConnectTimeout int                 `json:"service_connect_timeout"`
	ServiceWriteTimeout   int                 `json:"service_write_timeout"`
	ServiceReadTimeout    int                 `json:"service_read_timeout"`
	LegacyRetries         *int                `json:"retries,omitempty"`
	LegacyConnectTimeout  *int                `json:"connect_timeout,omitempty"`
	LegacyWriteTimeout    *int                `json:"write_timeout,omitempty"`
	LegacyReadTimeout     *int                `json:"read_timeout,omitempty"`
	AccessStripPath       bool                `json:"access_strip_path"`
	LegacyStripPath       *bool               `json:"strip_path,omitempty"`
	Plugins               []PluginSnapshot    `json:"plugins"`
}

// PluginSnapshot is the plugin binding captured when an API version is published.
type PluginSnapshot struct {
	Name    string                 `json:"name"`
	Plugin  string                 `json:"plugin"`
	Config  map[string]interface{} `json:"config,omitempty"`
	Enabled bool                   `json:"enabled"`
}

// EffectiveStripPath prefers the current field and falls back to snapshots stored as strip_path.
func (s APIConfigSnapshot) EffectiveStripPath() bool {
	if s.AccessStripPath {
		return true
	}
	if s.LegacyStripPath != nil {
		return *s.LegacyStripPath
	}
	return false
}

func (s APIConfigSnapshot) EffectiveProtocol() string {
	if s.ServiceProtocol != "" {
		return s.ServiceProtocol
	}
	return s.LegacyProtocol
}

func (s APIConfigSnapshot) EffectiveHostKind() string {
	if s.ServiceHostKind != "" {
		return s.ServiceHostKind
	}
	return s.LegacyHostKind
}

func (s APIConfigSnapshot) EffectiveHost() string {
	if s.ServiceHost != "" {
		return s.ServiceHost
	}
	return s.LegacyHost
}

func (s APIConfigSnapshot) EffectiveUpstreamID() uint64 {
	if s.ServiceUpstreamID != 0 {
		return s.ServiceUpstreamID
	}
	if s.LegacyUpstreamID != nil {
		return *s.LegacyUpstreamID
	}
	return 0
}

func (s APIConfigSnapshot) EffectivePort() int {
	return firstInt(s.ServicePort, s.LegacyPort)
}

func (s APIConfigSnapshot) EffectiveRetries() int {
	return firstInt(s.ServiceRetries, s.LegacyRetries)
}

func (s APIConfigSnapshot) EffectiveConnectTimeout() int {
	return firstInt(s.ServiceConnectTimeout, s.LegacyConnectTimeout)
}

func (s APIConfigSnapshot) EffectiveWriteTimeout() int {
	return firstInt(s.ServiceWriteTimeout, s.LegacyWriteTimeout)
}

func (s APIConfigSnapshot) EffectiveReadTimeout() int {
	return firstInt(s.ServiceReadTimeout, s.LegacyReadTimeout)
}

func firstInt(current int, legacy *int) int {
	if current != 0 {
		return current
	}
	if legacy != nil {
		return *legacy
	}
	return 0
}

type Upstream struct {
	ID                     uint64         `gorm:"primaryKey" json:"id"`
	SpaceID                uint64         `gorm:"not null;uniqueIndex:idx_space_upstream_name" json:"space_id"`
	Name                   string         `gorm:"size:128;not null;uniqueIndex:idx_space_upstream_name" json:"name"`
	Algorithm              string         `gorm:"size:32;not null;default:round-robin" json:"algorithm"`
	Slots                  int            `gorm:"not null;default:10000" json:"slots"`
	HashOn                 string         `gorm:"size:32" json:"hash_on"`
	HashFallback           string         `gorm:"size:32" json:"hash_fallback"`
	HashOnHeader           string         `gorm:"size:128" json:"hash_on_header"`
	HashFallbackHeader     string         `gorm:"size:128" json:"hash_fallback_header"`
	HashOnCookie           string         `gorm:"size:128" json:"hash_on_cookie"`
	HashOnCookiePath       string         `gorm:"size:128" json:"hash_on_cookie_path"`
	HashOnQueryArg         string         `gorm:"size:128" json:"hash_on_query_arg"`
	HashFallbackQueryArg   string         `gorm:"size:128" json:"hash_fallback_query_arg"`
	HashOnURICapture       string         `gorm:"size:128" json:"hash_on_uri_capture"`
	HashFallbackURICapture string         `gorm:"size:128" json:"hash_fallback_uri_capture"`
	Healthchecks           datatypes.JSON `gorm:"type:jsonb" json:"healthchecks"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`

	Targets  []UpstreamTarget  `gorm:"foreignKey:UpstreamID" json:"targets,omitempty"`
	Gateways []UpstreamGateway `gorm:"foreignKey:UpstreamID" json:"-"`
}

type UpstreamTarget struct {
	ID         uint64    `gorm:"primaryKey" json:"id"`
	UpstreamID uint64    `gorm:"not null;index" json:"upstream_id"`
	Target     string    `gorm:"size:255;not null" json:"target"`
	Weight     int       `gorm:"not null;default:100" json:"weight"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type UpstreamGateway struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	UpstreamID     uint64    `gorm:"not null;uniqueIndex:idx_upstream_gateway" json:"upstream_id"`
	GatewayID      uint64    `gorm:"not null;uniqueIndex:idx_upstream_gateway" json:"gateway_id"`
	KongUpstreamID string    `gorm:"size:64" json:"kong_upstream_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func KongUpstreamName(spaceID uint64, name string) string {
	return fmt.Sprintf("agm-s%d-%s", spaceID, name)
}

func KongConsumerName(spaceID uint64, username string) string {
	return fmt.Sprintf("agm-s%d-%s", spaceID, username)
}

// Consumer is a space-scoped Kong consumer.
type Consumer struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	SpaceID   uint64    `gorm:"not null;uniqueIndex:idx_space_consumer_username" json:"space_id"`
	Username  string    `gorm:"size:128;not null;uniqueIndex:idx_space_consumer_username" json:"username"`
	CustomID  string    `gorm:"size:128;not null;default:''" json:"custom_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Credentials []ConsumerCredential `gorm:"foreignKey:ConsumerID" json:"credentials,omitempty"`
	Gateways    []ConsumerGateway    `gorm:"foreignKey:ConsumerID" json:"-"`
	APIs        []API                `gorm:"many2many:api_consumers;" json:"apis,omitempty"`
	Space       *Space               `gorm:"foreignKey:SpaceID" json:"space,omitempty"`
}

// ConsumerCredential is a Kong credential or ACL group belonging to a consumer.
type ConsumerCredential struct {
	ID         uint64         `gorm:"primaryKey" json:"id"`
	ConsumerID uint64         `gorm:"not null;index" json:"consumer_id"`
	Plugin     string         `gorm:"size:32;not null" json:"plugin"`
	Config     datatypes.JSON `gorm:"type:jsonb;not null" json:"config"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// ConsumerGateway records the Kong consumer id on each gateway the space uses.
type ConsumerGateway struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	ConsumerID     uint64    `gorm:"not null;uniqueIndex:idx_consumer_gateway" json:"consumer_id"`
	GatewayID      uint64    `gorm:"not null;uniqueIndex:idx_consumer_gateway" json:"gateway_id"`
	KongConsumerID string    `gorm:"size:64" json:"kong_consumer_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SplitPaths returns normalized path list from a comma/newline separated string.
func SplitPaths(raw string) []string {
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
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

var prefixPattern = regexp.MustCompile(`^/[A-Za-z0-9][A-Za-z0-9_-]*(/[A-Za-z0-9][A-Za-z0-9_-]*)*$`)

// NormalizePrefix validates a space path prefix such as /order or /team/order.
func NormalizePrefix(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("prefix is required")
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	raw = strings.TrimRight(raw, "/")
	if !prefixPattern.MatchString(raw) {
		return "", fmt.Errorf("prefix must look like /order or /team/order")
	}
	return raw, nil
}

// ApplyPathPrefix prepends prefix to each path. Paths that already start with the prefix are kept.
func ApplyPathPrefix(prefix string, paths []string) []string {
	prefix = strings.TrimRight(strings.TrimSpace(prefix), "/")
	if prefix == "" || prefix == "/" {
		return paths
	}
	out := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, p := range paths {
		if p != prefix && !strings.HasPrefix(p, prefix+"/") {
			p = prefix + p
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}
