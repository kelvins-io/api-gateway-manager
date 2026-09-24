package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/response"
	"github.com/kelvins-io/api-gateway-manager/internal/service"
)

type UpstreamHandler struct {
	svc *service.UpstreamService
}

func NewUpstreamHandler(svc *service.UpstreamService) *UpstreamHandler {
	return &UpstreamHandler{svc: svc}
}

func (h *UpstreamHandler) List(c *gin.Context) {
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

func (h *UpstreamHandler) Create(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var in service.UpsertUpstreamInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	up, err := h.svc.Create(spaceID, in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, up)
}

func (h *UpstreamHandler) Update(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	id, ok := parseID(c, "upid")
	if !ok {
		return
	}
	if err := h.svc.EnsureSpace(id, spaceID); err != nil {
		mapError(c, err)
		return
	}
	var in service.UpsertUpstreamInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	up, err := h.svc.Update(c.Request.Context(), id, in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, up)
}

func (h *UpstreamHandler) Delete(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	id, ok := parseID(c, "upid")
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
