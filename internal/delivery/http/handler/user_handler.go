package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	mw "github.com/monir/auth_service/internal/middleware"
	"github.com/monir/auth_service/internal/domain/user"
	"github.com/monir/auth_service/internal/repository/postgres"
	"github.com/monir/auth_service/pkg/response"
)

type UserHandler struct {
	userRepo user.Repository
}

func NewUserHandler(userRepo user.Repository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

// GET /api/v1/users/me
func (h *UserHandler) Me(c *gin.Context) {
	claims := mw.ClaimsFromContext(c)
	if claims == nil {
		response.Unauthorized(c, "not authenticated")
		return
	}
	u, err := h.userRepo.FindByID(c.Request.Context(), claims.UserID)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, toUserResponse(u))
}

// GET /api/v1/users/:id
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	u, err := h.userRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, toUserResponse(u))
}

// PUT /api/v1/users/:id
func (h *UserHandler) Update(c *gin.Context) {
	claims := mw.ClaimsFromContext(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	// Users can only update themselves unless they have admin role
	if claims != nil && claims.UserID != id && !hasRole(claims.Roles, "admin") {
		response.Forbidden(c, "cannot update another user")
		return
	}

	var body struct {
		Phone string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	u, err := h.userRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		response.InternalError(c)
		return
	}
	if body.Phone != "" {
		u.Phone = body.Phone
	}
	if err := h.userRepo.Update(c.Request.Context(), u); err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, toUserResponse(u))
}

// DELETE /api/v1/users/:id
func (h *UserHandler) Delete(c *gin.Context) {
	claims := mw.ClaimsFromContext(c)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	if claims != nil && claims.UserID != id && !hasRole(claims.Roles, "admin") {
		response.Forbidden(c, "cannot delete another user")
		return
	}
	if err := h.userRepo.SoftDelete(c.Request.Context(), id); err != nil {
		response.InternalError(c)
		return
	}
	response.NoContent(c)
}

type userResponse struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	Phone         string    `json:"phone"`
	EmailVerified bool      `json:"email_verified"`
	Status        string    `json:"status"`
}

func toUserResponse(u *user.User) userResponse {
	return userResponse{
		ID:            u.ID,
		Email:         u.Email,
		Phone:         u.Phone,
		EmailVerified: u.EmailVerified,
		Status:        string(u.Status),
	}
}

func hasRole(roles []string, target string) bool {
	for _, r := range roles {
		if r == target {
			return true
		}
	}
	return false
}
