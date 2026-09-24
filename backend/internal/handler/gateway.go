package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/response"
	"github.com/kelvins-io/api-gateway-manager/internal/service"
)

type GatewayHandler struct {
	svc *service.GatewayService
}

func NewGatewayHandler(svc *service.GatewayService) *GatewayHandler {
	return &GatewayHandler{svc: svc}
}

func (h *GatewayHandler) Create(c *gin.Context) {
	var in service.CreateGatewayInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	gw, err := h.svc.Create(in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, gw)
}

func (h *GatewayHandler) List(c *gin.Context) {
	list, err := h.svc.List()
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *GatewayHandler) ListOptions(c *gin.Context) {
	list, err := h.svc.ListOptions()
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *GatewayHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	gw, err := h.svc.Get(id)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, gw)
}

func (h *GatewayHandler) Update(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var in service.UpdateGatewayInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	gw, err := h.svc.Update(id, in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, gw)
}

func (h *GatewayHandler) Delete(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, nil)
}

type probeGatewayInput struct {
	AdminAPI string `json:"admin_api" binding:"required,min=8,max=512"`
}

func (h *GatewayHandler) Probe(c *gin.Context) {
	var in probeGatewayInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.Probe(in.AdminAPI); err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, gin.H{"ok": true})
}
