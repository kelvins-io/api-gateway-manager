package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/response"
	"github.com/kelvins-io/api-gateway-manager/internal/service"
)

type APIHandler struct {
	svc *service.APIService
}

func NewAPIHandler(svc *service.APIService) *APIHandler {
	return &APIHandler{svc: svc}
}

func (h *APIHandler) Create(c *gin.Context) {
	groupID, ok := parseID(c, "gid")
	if !ok {
		return
	}
	var in service.CreateAPIInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	api, err := h.svc.Create(groupID, in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, api)
}

func (h *APIHandler) List(c *gin.Context) {
	groupID, ok := parseID(c, "gid")
	if !ok {
		return
	}
	list, err := h.svc.ListByGroup(groupID)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *APIHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "aid")
	if !ok {
		return
	}
	api, err := h.svc.Get(id)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, api)
}

func (h *APIHandler) Update(c *gin.Context) {
	id, ok := parseID(c, "aid")
	if !ok {
		return
	}
	var in service.UpdateAPIInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	api, err := h.svc.Update(id, in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, api)
}

func (h *APIHandler) Delete(c *gin.Context) {
	id, ok := parseID(c, "aid")
	if !ok {
		return
	}
	if err := h.svc.Delete(id); err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *APIHandler) Publish(c *gin.Context) {
	id, ok := parseID(c, "aid")
	if !ok {
		return
	}
	api, err := h.svc.Publish(c.Request.Context(), id)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, api)
}

func (h *APIHandler) Offline(c *gin.Context) {
	id, ok := parseID(c, "aid")
	if !ok {
		return
	}
	api, err := h.svc.Offline(c.Request.Context(), id)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, api)
}

func (h *APIHandler) SwitchVersion(c *gin.Context) {
	id, ok := parseID(c, "aid")
	if !ok {
		return
	}
	var in service.SwitchVersionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	api, err := h.svc.SwitchVersion(c.Request.Context(), id, in.Version)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, api)
}

func (h *APIHandler) ListVersions(c *gin.Context) {
	id, ok := parseID(c, "aid")
	if !ok {
		return
	}
	list, err := h.svc.ListVersions(id)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, list)
}
