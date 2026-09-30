package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/middleware"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/response"
	"github.com/kelvins-io/api-gateway-manager/internal/service"
	"gorm.io/gorm"
)

type DebugHandler struct {
	proxy   *service.DebugProxyService
	history *service.DebugHistoryService
	db      *gorm.DB
}

func NewDebugHandler(proxy *service.DebugProxyService, history *service.DebugHistoryService, db *gorm.DB) *DebugHandler {
	return &DebugHandler{proxy: proxy, history: history, db: db}
}

func (h *DebugHandler) Proxy(c *gin.Context) {
	var in service.DebugProxyInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.proxy.Proxy(c.Request.Context(), in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *DebugHandler) ListHistories(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	list, err := h.history.List(spaceID)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *DebugHandler) CreateHistory(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var in service.CreateDebugHistoryInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item, err := h.history.Create(spaceID, middleware.GetUserID(c), in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *DebugHandler) GetHistory(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	hid, ok := parseID(c, "hid")
	if !ok {
		return
	}
	item, err := h.history.Get(spaceID, hid)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *DebugHandler) DeleteHistory(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	hid, ok := parseID(c, "hid")
	if !ok {
		return
	}
	userID := middleware.GetUserID(c)
	isAdmin := middleware.GetRole(c) == model.RoleSystemAdmin || h.isSpaceAdmin(spaceID, userID)
	if err := h.history.Delete(spaceID, hid, userID, isAdmin); err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *DebugHandler) isSpaceAdmin(spaceID, userID uint64) bool {
	var member model.SpaceMember
	err := h.db.Where(
		"space_id = ? AND user_id = ? AND status = ? AND role = ?",
		spaceID, userID, model.MemberStatusActive, model.RoleSpaceAdmin,
	).First(&member).Error
	return err == nil
}
