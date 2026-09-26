package router

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/handler"
	"github.com/kelvins-io/api-gateway-manager/internal/middleware"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/jwtutil"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/response"
	"github.com/kelvins-io/api-gateway-manager/internal/service"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Handlers struct {
	Auth     *handler.AuthHandler
	Space    *handler.SpaceHandler
	Gateway  *handler.GatewayHandler
	Group    *handler.GroupHandler
	API      *handler.APIHandler
	Upstream *handler.UpstreamHandler
	Consumer *handler.ConsumerHandler
	Plugin   *handler.PluginHandler
	APISvc   *service.APIService
	GroupSvc *service.GroupService
}

func Setup(db *gorm.DB, jwtMgr *jwtutil.Manager, log *zap.Logger, h Handlers) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(log))
	r.Use(cors())

	r.GET("/health", func(c *gin.Context) {
		response.OK(c, gin.H{"status": "up"})
	})

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", h.Auth.Register)
			auth.POST("/login", h.Auth.Login)
			auth.GET("/me", middleware.JWTAuth(jwtMgr), h.Auth.Me)
		}

		authed := v1.Group("")
		authed.Use(middleware.JWTAuth(jwtMgr))
		{
			// spaces
			authed.GET("/spaces", h.Space.List)
			authed.GET("/spaces/available", h.Space.ListAvailable)
			authed.POST("/spaces", h.Space.Create)
			authed.GET("/spaces/:id", middleware.RequireSpaceRole(db, "id"), h.Space.Get)
			authed.PUT("/spaces/:id", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Space.Update)
			authed.DELETE("/spaces/:id", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Space.Delete)
			authed.POST("/spaces/:id/approve", middleware.RequireSystemAdmin(), h.Space.Approve)
			authed.POST("/spaces/:id/reject", middleware.RequireSystemAdmin(), h.Space.Reject)
			authed.POST("/spaces/:id/join", h.Space.Join)
			authed.GET("/spaces/:id/members", middleware.RequireSpaceRole(db, "id"), h.Space.ListMembers)
			authed.GET("/spaces/:id/members/candidates", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Space.ListCandidateUsers)
			authed.POST("/spaces/:id/members", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Space.AddMember)
			authed.POST("/spaces/:id/members/:uid/approve", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Space.ApproveMember)
			authed.POST("/spaces/:id/members/:uid/reject", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Space.RejectMember)
			authed.DELETE("/spaces/:id/members/:uid", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Space.RemoveMember)
			authed.PUT("/spaces/:id/members/:uid/role", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Space.UpdateMemberRole)

			// groups under space
			authed.GET("/spaces/:id/groups", middleware.RequireSpaceRole(db, "id"), h.Group.List)
			authed.POST("/spaces/:id/groups", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Group.Create)
			authed.GET("/spaces/:id/gateway-options", middleware.RequireSpaceRole(db, "id"), h.Gateway.ListOptionsForSpace)
			authed.GET("/spaces/:id/upstreams", middleware.RequireSpaceRole(db, "id"), h.Upstream.List)
			authed.POST("/spaces/:id/upstreams", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Upstream.Create)
			authed.PUT("/spaces/:id/upstreams/:upid", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Upstream.Update)
			authed.DELETE("/spaces/:id/upstreams/:upid", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Upstream.Delete)
			authed.GET("/spaces/:id/apis", middleware.RequireSpaceRole(db, "id"), h.API.ListBySpace)
			authed.GET("/spaces/:id/consumers", middleware.RequireSpaceRole(db, "id"), h.Consumer.List)
			authed.POST("/spaces/:id/consumers", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Consumer.Create)
			authed.POST("/spaces/:id/consumers/sync", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Consumer.Sync)
			authed.PUT("/spaces/:id/consumers/:cid", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Consumer.Update)
			authed.DELETE("/spaces/:id/consumers/:cid", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Consumer.Delete)
			authed.GET("/spaces/:id/plugins", middleware.RequireSpaceRole(db, "id"), h.Plugin.List)
			authed.GET("/spaces/:id/plugins/:pid/apis", middleware.RequireSpaceRole(db, "id"), h.Plugin.ListAPIs)
			authed.POST("/spaces/:id/plugins", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Plugin.Create)
			authed.PUT("/spaces/:id/plugins/:pid", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Plugin.Update)
			authed.DELETE("/spaces/:id/plugins/:pid", middleware.RequireSpaceRole(db, "id", model.RoleSpaceAdmin), h.Plugin.Delete)

			// gateways: full CRUD only for system admin
			admin := authed.Group("")
			admin.Use(middleware.RequireSystemAdmin())
			{
				admin.GET("/gateways", h.Gateway.List)
				admin.POST("/gateways/probe", h.Gateway.Probe)
				admin.GET("/gateways/:id", h.Gateway.Get)
				admin.POST("/gateways", h.Gateway.Create)
				admin.PUT("/gateways/:id", h.Gateway.Update)
				admin.DELETE("/gateways/:id", h.Gateway.Delete)
				admin.GET("/gateways/:id/unauthorized-spaces", h.Gateway.ListUnauthorizedSpaces)
				admin.POST("/gateways/:id/authorize-spaces", h.Gateway.AuthorizeSpaces)
			}
			// options for binding api groups (no admin_api exposed); prefers space-scoped route
			authed.GET("/gateways-options", h.Gateway.ListOptions)

			// group detail / update / delete
			authed.GET("/groups/:gid", requireGroupAccess(db, h.GroupSvc, false), h.Group.Get)
			authed.PUT("/groups/:gid", requireGroupAccess(db, h.GroupSvc, true), h.Group.Update)
			authed.DELETE("/groups/:gid", requireGroupAccess(db, h.GroupSvc, true), h.Group.Delete)

			// apis under group
			authed.GET("/groups/:gid/apis", requireGroupAccess(db, h.GroupSvc, false), h.API.List)
			authed.POST("/groups/:gid/apis", requireGroupAccess(db, h.GroupSvc, true), h.API.Create)
			authed.POST("/groups/:gid/apis/import-openapi", requireGroupAccess(db, h.GroupSvc, true), h.API.ImportOpenAPI)

			authed.GET("/market/apis", h.API.ListMarket)
			authed.GET("/market/apis/:aid/linkable-consumers", h.API.ListMarketLinkableConsumers)
			authed.POST("/market/apis/:aid/link-consumers", h.API.LinkMarketConsumers)

			authed.GET("/apis/:aid", requireAPIAccess(db, h.APISvc, false), h.API.Get)
			authed.PUT("/apis/:aid", requireAPIAccess(db, h.APISvc, true), h.API.Update)
			authed.DELETE("/apis/:aid", requireAPIAccess(db, h.APISvc, true), h.API.Delete)
			authed.POST("/apis/:aid/publish", requireAPIAccess(db, h.APISvc, true), h.API.Publish)
			authed.POST("/apis/:aid/offline", requireAPIAccess(db, h.APISvc, true), h.API.Offline)
			authed.POST("/apis/:aid/share", requireAPIAccess(db, h.APISvc, true), h.API.Share)
			authed.POST("/apis/:aid/unshare", requireAPIAccess(db, h.APISvc, true), h.API.Unshare)
			authed.POST("/apis/:aid/switch-version", requireAPIAccess(db, h.APISvc, true), h.API.SwitchVersion)
			authed.GET("/apis/:aid/versions", requireAPIAccess(db, h.APISvc, false), h.API.ListVersions)
		}
	}

	return r
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func requireGroupAccess(db *gorm.DB, groupSvc *service.GroupService, write bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if middleware.GetRole(c) == model.RoleSystemAdmin {
			c.Next()
			return
		}
		gid, err := parseUintParam(c, "gid")
		if err != nil {
			response.BadRequest(c, "invalid gid")
			c.Abort()
			return
		}
		spaceID, err := groupSvc.SpaceIDOfGroup(gid)
		if err != nil {
			response.NotFound(c, "group not found")
			c.Abort()
			return
		}
		checkSpaceMember(c, db, spaceID, write)
	}
}

func requireAPIAccess(db *gorm.DB, apiSvc *service.APIService, write bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if middleware.GetRole(c) == model.RoleSystemAdmin {
			c.Next()
			return
		}
		aid, err := parseUintParam(c, "aid")
		if err != nil {
			response.BadRequest(c, "invalid aid")
			c.Abort()
			return
		}
		spaceID, err := apiSvc.SpaceIDOfAPI(aid)
		if err != nil {
			response.NotFound(c, "api not found")
			c.Abort()
			return
		}
		checkSpaceMember(c, db, spaceID, write)
	}
}

func checkSpaceMember(c *gin.Context, db *gorm.DB, spaceID uint64, write bool) {
	userID := middleware.GetUserID(c)
	var space model.Space
	if err := db.First(&space, spaceID).Error; err != nil {
		response.NotFound(c, "space not found")
		c.Abort()
		return
	}
	if space.Status != model.SpaceStatusActive {
		response.Forbidden(c, "space is not active")
		c.Abort()
		return
	}
	var member model.SpaceMember
	if err := db.Where("space_id = ? AND user_id = ? AND status = ?", spaceID, userID, model.MemberStatusActive).First(&member).Error; err != nil {
		response.Forbidden(c, "not a member of this space")
		c.Abort()
		return
	}
	if write && member.Role != model.RoleSpaceAdmin {
		response.Forbidden(c, "space admin required")
		c.Abort()
		return
	}
	c.Next()
}

func parseUintParam(c *gin.Context, name string) (uint64, error) {
	return strconv.ParseUint(c.Param(name), 10, 64)
}
