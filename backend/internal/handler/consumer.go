package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/response"
	"github.com/kelvins-io/api-gateway-manager/internal/service"
)

type ConsumerHandler struct {
	svc *service.ConsumerService
}

func NewConsumerHandler(svc *service.ConsumerService) *ConsumerHandler {
	return &ConsumerHandler{svc: svc}
}

func (h *ConsumerHandler) List(c *gin.Context) {
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

func (h *ConsumerHandler) Create(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var in service.UpsertConsumerInput
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

func (h *ConsumerHandler) Update(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	id, ok := parseID(c, "cid")
	if !ok {
		return
	}
	if err := h.svc.EnsureSpace(id, spaceID); err != nil {
		mapError(c, err)
		return
	}
	var in service.UpsertConsumerInput
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

func (h *ConsumerHandler) Delete(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	id, ok := parseID(c, "cid")
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

func (h *ConsumerHandler) Sync(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.ReconcileSpace(c.Request.Context(), spaceID); err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, nil)
}
