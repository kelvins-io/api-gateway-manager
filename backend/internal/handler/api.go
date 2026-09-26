package handler

import (
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/middleware"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
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

func (h *APIHandler) ImportOpenAPI(c *gin.Context) {
	groupID, ok := parseID(c, "gid")
	if !ok {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "please upload an OpenAPI JSON or YAML file")
		return
	}
	f, err := file.Open()
	if err != nil {
		response.BadRequest(c, "failed to open uploaded file")
		return
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, 8<<20)) // 8 MiB
	if err != nil {
		response.BadRequest(c, "failed to read uploaded file")
		return
	}
	port := 0
	if raw := strings.TrimSpace(c.PostForm("service_port")); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil {
			response.BadRequest(c, "invalid service_port")
			return
		}
		port = p
	}
	dryRun := false
	switch strings.ToLower(strings.TrimSpace(c.PostForm("dry_run"))) {
	case "1", "true", "yes":
		dryRun = true
	}
	result, err := h.svc.ImportOpenAPI(c.Request.Context(), groupID, service.ImportOpenAPIOptions{
		Content:         content,
		Filename:        file.Filename,
		DryRun:          dryRun,
		ServiceProtocol: c.PostForm("service_protocol"),
		ServiceHost:     c.PostForm("service_host"),
		ServicePort:     port,
		ServicePath:     c.PostForm("service_path"),
	})
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *APIHandler) ListBySpace(c *gin.Context) {
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
	api, err := h.svc.Update(c.Request.Context(), id, in)
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

func (h *APIHandler) Share(c *gin.Context) {
	id, ok := parseID(c, "aid")
	if !ok {
		return
	}
	api, err := h.svc.Share(id)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, api)
}

func (h *APIHandler) Unshare(c *gin.Context) {
	id, ok := parseID(c, "aid")
	if !ok {
		return
	}
	api, err := h.svc.Unshare(id)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, api)
}

func (h *APIHandler) ListMarket(c *gin.Context) {
	list, err := h.svc.ListMarket()
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *APIHandler) ListMarketLinkableConsumers(c *gin.Context) {
	id, ok := parseID(c, "aid")
	if !ok {
		return
	}
	list, err := h.svc.ListMarketLinkableConsumers(middleware.GetUserID(c), middleware.GetRole(c) == model.RoleSystemAdmin, id)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *APIHandler) LinkMarketConsumers(c *gin.Context) {
	id, ok := parseID(c, "aid")
	if !ok {
		return
	}
	var in service.LinkMarketConsumersInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.LinkMarketConsumers(c.Request.Context(), middleware.GetUserID(c), middleware.GetRole(c) == model.RoleSystemAdmin, id, in.ConsumerIDs); err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, nil)
}
