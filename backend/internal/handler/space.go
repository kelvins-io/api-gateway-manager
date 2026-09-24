package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/middleware"
	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/response"
	"github.com/kelvins-io/api-gateway-manager/internal/service"
)

type SpaceHandler struct {
	svc *service.SpaceService
}

func NewSpaceHandler(svc *service.SpaceService) *SpaceHandler {
	return &SpaceHandler{svc: svc}
}

func (h *SpaceHandler) Create(c *gin.Context) {
	var in service.CreateSpaceInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	space, err := h.svc.Create(middleware.GetUserID(c), in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, space)
}

func (h *SpaceHandler) List(c *gin.Context) {
	isAdmin := middleware.GetRole(c) == model.RoleSystemAdmin
	list, err := h.svc.List(middleware.GetUserID(c), isAdmin)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *SpaceHandler) ListAvailable(c *gin.Context) {
	list, err := h.svc.ListAvailable(middleware.GetUserID(c))
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, list)
}

func (h *SpaceHandler) Get(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	space, err := h.svc.Get(id)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, space)
}

func (h *SpaceHandler) Update(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var in service.UpdateSpaceInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	space, err := h.svc.Update(id, in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, space)
}

func (h *SpaceHandler) Delete(c *gin.Context) {
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

func (h *SpaceHandler) Join(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Join(id, middleware.GetUserID(c)); err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *SpaceHandler) ListMembers(c *gin.Context) {
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	members, err := h.svc.ListMembers(id)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, members)
}

func (h *SpaceHandler) UpdateMemberRole(c *gin.Context) {
	spaceID, ok := parseID(c, "id")
	if !ok {
		return
	}
	uid, ok := parseID(c, "uid")
	if !ok {
		return
	}
	var in service.UpdateMemberRoleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.UpdateMemberRole(spaceID, uid, in.Role); err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, nil)
}
