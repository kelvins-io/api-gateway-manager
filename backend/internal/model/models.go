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

	Gateway *Gateway `gorm:"foreignKey:GatewayID" json:"gateway,omitempty"`
}

type API struct {
	ID              uint64    `gorm:"primaryKey" json:"id"`
	GroupID         uint64    `gorm:"not null;index" json:"group_id"`
	Name            string    `gorm:"size:128;not null" json:"name"`
	Path            string    `gorm:"size:2048;not null" json:"path"` // comma-separated, e.g. /a,/b
	Methods         string    `gorm:"size:128;not null" json:"methods"`
	AccessProtocols string    `gorm:"size:64;not null;default:http" json:"access_protocols"`
	UpstreamURL     string    `gorm:"size:512" json:"upstream_url"`
	Protocol        string    `gorm:"size:16;not null;default:http" json:"protocol"`
	HostKind        string    `gorm:"size:16;not null;default:direct" json:"host_kind"` // direct | upstream
	Host            string    `gorm:"size:255" json:"host"`
	UpstreamID      *uint64   `gorm:"index" json:"upstream_id,omitempty"`
	Port            int       `gorm:"not null;default:80" json:"port"`
	ServicePath     string    `gorm:"size:512;not null;default:/" json:"service_path"`
	Retries         int       `gorm:"not null;default:5" json:"retries"`
	ConnectTimeout  int       `gorm:"not null;default:60000" json:"connect_timeout"`
	WriteTimeout    int       `gorm:"not null;default:60000" json:"write_timeout"`
	ReadTimeout     int       `gorm:"not null;default:60000" json:"read_timeout"`
	StripPath       bool      `gorm:"default:true" json:"strip_path"`
	Status          string    `gorm:"size:32;not null;default:draft" json:"status"`
	CurrentVersion  string    `gorm:"size:32" json:"current_version"`
	KongServiceID   string    `gorm:"size:64" json:"kong_service_id"`
	KongRouteID     string    `gorm:"size:64" json:"kong_route_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Group *APIGroup `gorm:"foreignKey:GroupID" json:"group,omitempty"`
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
	Name            string `json:"name"`
	Path            string `json:"path"` // comma-separated route paths
	Methods         string `json:"methods"`
	AccessProtocols string `json:"access_protocols"`
	UpstreamURL     string `json:"upstream_url,omitempty"`
	Protocol        string `json:"protocol"`
	HostKind        string `json:"host_kind"`
	Host            string `json:"host"`
	UpstreamID      uint64 `json:"upstream_id,omitempty"`
	KongHost        string `json:"kong_host"`
	Port            int    `json:"port"`
	ServicePath     string `json:"service_path"`
	Retries         int    `json:"retries"`
	ConnectTimeout  int    `json:"connect_timeout"`
	WriteTimeout    int    `json:"write_timeout"`
	ReadTimeout     int    `json:"read_timeout"`
	StripPath       bool   `json:"strip_path"`
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
