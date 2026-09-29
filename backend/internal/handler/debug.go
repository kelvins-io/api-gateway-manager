package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/response"
	"github.com/kelvins-io/api-gateway-manager/internal/service"
)

type DebugHandler struct {
	svc *service.DebugProxyService
}

func NewDebugHandler(svc *service.DebugProxyService) *DebugHandler {
	return &DebugHandler{svc: svc}
}

func (h *DebugHandler) Proxy(c *gin.Context) {
	var in service.DebugProxyInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.Proxy(c.Request.Context(), in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, result)
}
