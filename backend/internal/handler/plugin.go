package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/response"
	"github.com/kelvins-io/api-gateway-manager/internal/service"
)

type PluginHandler struct {
	svc *service.PluginService
}

func NewPluginHandler(svc *service.PluginService) *PluginHandler {
	return &PluginHandler{svc: svc}
}

func (h *PluginHandler) List(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	list, err := h.svc.List(spaceID)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *PluginHandler) ListAPIs(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	id, ok := parseID(c, "pid")
	if !ok {
		return
	}
	list, err := h.svc.ListAPIs(id, spaceID)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *PluginHandler) Create(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var in service.UpsertPluginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item, err := h.svc.Create(c.Request.Context(), spaceID, in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *PluginHandler) Update(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	id, ok := parseID(c, "pid")
	if !ok {
		return
	}
	if err := h.svc.EnsureSpace(id, spaceID); err != nil {
		mapError(c, err)
		return
	}
	var in service.UpsertPluginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	item, err := h.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, item)
}

func (h *PluginHandler) Delete(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	id, ok := parseID(c, "pid")
	if !ok {
		return
	}
	if err := h.svc.EnsureSpace(id, spaceID); err != nil {
		mapError(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, nil)
}
