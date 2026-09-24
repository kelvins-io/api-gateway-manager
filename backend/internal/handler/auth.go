package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/kelvins-io/api-gateway-manager/internal/middleware"
	"github.com/kelvins-io/api-gateway-manager/internal/pkg/response"
	"github.com/kelvins-io/api-gateway-manager/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var in service.RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.Register(in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var in service.LoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.Login(in)
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, err := h.svc.GetUser(middleware.GetUserID(c))
	if err != nil {
		mapError(c, err)
		return
	}
	response.OK(c, user)
}

func mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		response.Unauthorized(c, err.Error())
	case errors.Is(err, service.ErrForbidden):
		response.Forbidden(c, err.Error())
	case errors.Is(err, service.ErrNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, service.ErrConflict):
		response.Fail(c, 409, 409, err.Error())
	case errors.Is(err, service.ErrBadRequest):
		response.BadRequest(c, err.Error())
	default:
		response.ServerError(c, err.Error())
	}
}

func parseID(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid "+name)
		return 0, false
	}
	return id, true
}
