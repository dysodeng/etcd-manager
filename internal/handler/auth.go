package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dysodeng/etcd-manager/internal/service"
)

type AuthHandler struct {
	authSvc *service.AuthService
	userSvc *service.UserService
}

func NewAuthHandler(authSvc *service.AuthService, userSvc *service.UserService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, userSvc: userSvc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, "username and password required")
		return
	}
	result, err := h.authSvc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		Fail(c, CodeAuthFailed, err.Error())
		return
	}
	OK(c, result)
}

func (h *AuthHandler) Profile(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	user, err := h.userSvc.GetByID(c.Request.Context(), userID)
	if err != nil {
		Fail(c, CodeUnauthorized, "user not found")
		return
	}
	OK(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	OK(c, nil)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, CodeParamInvalid, "old_password and new_password required")
		return
	}
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	if err := h.authSvc.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		Fail(c, CodeAuthFailed, err.Error())
		return
	}
	OK(c, nil)
}

// getUserID extracts and parses the user_id from gin context as uuid.UUID.
func getUserID(c *gin.Context) (uuid.UUID, bool) {
	raw, _ := c.Get("user_id")
	s, ok := raw.(string)
	if !ok {
		Fail(c, CodeUnauthorized, "invalid user identity")
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		Fail(c, CodeUnauthorized, "invalid user identity")
		return uuid.UUID{}, false
	}
	return id, true
}

// parseUUID parses a raw string as uuid.UUID, sending an error response on failure.
func parseUUID(c *gin.Context, raw, paramName string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		Fail(c, CodeParamInvalid, "invalid "+paramName)
		return uuid.UUID{}, false
	}
	return id, true
}
