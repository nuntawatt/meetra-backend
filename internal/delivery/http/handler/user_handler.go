package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-wego/wego/internal/usecase/user"
	"github.com/go-wego/wego/pkg/response"
	"github.com/go-wego/wego/pkg/validator"
	"github.com/google/uuid"
)

// UserHandler handles HTTP requests for user operations.
type UserHandler struct {
	uc user.UseCase
}

// NewUserHandler constructs a UserHandler.
func NewUserHandler(uc user.UseCase) *UserHandler {
	return &UserHandler{uc: uc}
}

// ——— Register ————————————————————————————————————————————————————————————————

// Register godoc
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body user.RegisterInput true "Registration payload"
// @Success      201  {object} entity.User
// @Router       /api/v1/auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var in user.RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid JSON body")
		return
	}
	if err := validator.Validate(in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	u, err := h.uc.Register(c.Request.Context(), in)
	if err != nil {
		switch {
		case errors.Is(err, user.ErrEmailTaken):
			response.Conflict(c, err.Error())
		default:
			response.InternalError(c, "registration failed")
		}
		return
	}

	response.Created(c, u)
}

// ——— Login ———————————————————————————————————————————————————————————————————

// Login godoc
// @Summary      Login and receive JWT tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body user.LoginInput true "Login credentials"
// @Success      200  {object} user.AuthResponse
// @Router       /api/v1/auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var in user.LoginInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid JSON body")
		return
	}
	if err := validator.Validate(in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authResp, err := h.uc.Login(c.Request.Context(), in)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
			response.Unauthorized(c, err.Error())
			return
		}
		response.InternalError(c, "login failed")
		return
	}

	response.OK(c, authResp)
}

// ——— GetProfile ——————————————————————————————————————————————————————————————

// GetProfile godoc
// @Summary      Get current user profile
// @Tags         users
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} entity.User
// @Router       /api/v1/users/me [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		response.Unauthorized(c, "invalid user context")
		return
	}

	u, err := h.uc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			response.NotFound(c, err.Error())
			return
		}
		response.InternalError(c, "failed to get profile")
		return
	}

	response.OK(c, u)
}

// ——— UpdateProfile ———————————————————————————————————————————————————————————

// UpdateProfile godoc
// @Summary      Update current user profile
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body body user.UpdateInput true "Update payload"
// @Success      200  {object} entity.User
// @Router       /api/v1/users/me [patch]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		response.Unauthorized(c, "invalid user context")
		return
	}

	var in user.UpdateInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.BadRequest(c, "invalid JSON body")
		return
	}
	if err := validator.Validate(in); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	u, err := h.uc.UpdateProfile(c.Request.Context(), userID, in)
	if err != nil {
		response.InternalError(c, "failed to update profile")
		return
	}

	response.OK(c, u)
}

// ——— DeleteUser ——————————————————————————————————————————————————————————————

// DeleteUser godoc
// @Summary      Delete current user account
// @Tags         users
// @Security     BearerAuth
// @Produce      json
// @Success      204
// @Router       /api/v1/users/me [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		response.Unauthorized(c, "invalid user context")
		return
	}

	if err := h.uc.DeleteUser(c.Request.Context(), userID); err != nil {
		response.InternalError(c, "failed to delete account")
		return
	}

	response.NoContent(c)
}

// ——— Helpers —————————————————————————————————————————————————————————————————

// extractUserID parses the user_id set by the Auth middleware.
func extractUserID(c *gin.Context) (uuid.UUID, error) {
	return uuid.Parse(c.GetString("user_id"))
}
