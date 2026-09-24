package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/jwtutil"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	ContextUserID   = "user_id"
	ContextUsername = "username"
	ContextRole     = "role"
)

func JWTAuth(jwtMgr *jwtutil.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			response.Unauthorized(c, "missing or invalid authorization header")
			c.Abort()
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		claims, err := jwtMgr.Parse(token)
		if err != nil {
			response.Unauthorized(c, "invalid token")
			c.Abort()
			return
		}
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUsername, claims.Username)
		c.Set(ContextRole, claims.Role)
		c.Next()
	}
}

func RequireSystemAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(ContextRole)
		if role != model.RoleSystemAdmin {
			response.Forbidden(c, "system admin required")
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireSpaceRole ensures the current user is a member of the space with at least one of the allowed roles.
// System admins always pass. spaceIDParam is the path param name for space id.
func RequireSpaceRole(db *gorm.DB, spaceIDParam string, allowedRoles ...string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role, _ := c.Get(ContextRole)
		if role == model.RoleSystemAdmin {
			c.Next()
			return
		}

		spaceID := c.Param(spaceIDParam)
		if spaceID == "" {
			response.BadRequest(c, "space id required")
			c.Abort()
			return
		}

		userID, _ := c.Get(ContextUserID)
		var member model.SpaceMember
		err := db.Where("space_id = ? AND user_id = ?", spaceID, userID).First(&member).Error
		if err != nil {
			response.Forbidden(c, "not a member of this space")
			c.Abort()
			return
		}
		if len(allowed) > 0 {
			if _, ok := allowed[member.Role]; !ok {
				response.Forbidden(c, "insufficient space permission")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func RequestLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		log.Info("http request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.String("client_ip", c.ClientIP()),
		)
	}
}

func GetUserID(c *gin.Context) uint64 {
	v, _ := c.Get(ContextUserID)
	id, _ := v.(uint64)
	return id
}

func GetRole(c *gin.Context) string {
	v, _ := c.Get(ContextRole)
	role, _ := v.(string)
	return role
}
