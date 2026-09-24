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

	APIStatusDraft    = "draft"
	APIStatusPublished = "published"
	APIStatusOffline  = "offline"
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
	ID             uint64    `gorm:"primaryKey" json:"id"`
	GroupID        uint64    `gorm:"not null;index" json:"group_id"`
	Name           string    `gorm:"size:128;not null" json:"name"`
	Path           string    `gorm:"size:2048;not null" json:"path"` // comma-separated, e.g. /a,/b
	Methods        string    `gorm:"size:128;not null" json:"methods"`
	UpstreamURL    string    `gorm:"size:512;not null" json:"upstream_url"`
	StripPath      bool      `gorm:"default:true" json:"strip_path"`
	Status         string    `gorm:"size:32;not null;default:draft" json:"status"`
	CurrentVersion string    `gorm:"size:32" json:"current_version"`
	KongServiceID  string    `gorm:"size:64" json:"kong_service_id"`
	KongRouteID    string    `gorm:"size:64" json:"kong_route_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

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
	Name        string `json:"name"`
	Path        string `json:"path"` // comma-separated paths
	Methods     string `json:"methods"`
	UpstreamURL string `json:"upstream_url"`
	StripPath   bool   `json:"strip_path"`
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
