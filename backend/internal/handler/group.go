package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/response"
	"github.com/kelvins-io/api-gateway-manager/internal/service"
)

type GroupHandler struct {
	svc       *service.GroupService
	consumers *service.ConsumerService
}

func NewGroupHandler(svc *service.GroupService, consumers *service.ConsumerService) *GroupHandler {
	return &GroupHandler{svc: svc, consumers: consumers}
}

func (h *GroupHandler) Create(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var in service.CreateGroupInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	group, err := h.svc.Create(spaceID, in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, group)
}

func (h *GroupHandler) List(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	list, err := h.svc.ListBySpace(spaceID)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *GroupHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "gid")
	if !ok {
		return
	}
	group, err := h.svc.Get(id)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, group)
}

func (h *GroupHandler) Update(c *gin.Context) {
	id, ok := parseID(c, "gid")
	if !ok {
		return
	}
	var in service.UpdateGroupInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	group, err := h.svc.Update(id, in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, group)
}

func (h *GroupHandler) Delete(c *gin.Context) {
	id, ok := parseID(c, "gid")
	if !ok {
		return
	}
	group, err := h.svc.Get(id)
	if err != nil {
		mapError(c, err)
		return
	}
	if err := h.svc.Delete(id); err != nil {
		mapError(c, err)
		return
	}
	if err := h.consumers.ReleaseGateway(c.Request.Context(), group.SpaceID, group.GatewayID); err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, nil)
}
