package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/monir/auth_service/internal/domain/permission"
	"github.com/monir/auth_service/internal/domain/role"
	"github.com/monir/auth_service/internal/repository/postgres"
	"github.com/monir/auth_service/pkg/response"
)

type RoleHandler struct {
	roleRepo role.Repository
	permRepo permission.Repository
}

func NewRoleHandler(roleRepo role.Repository, permRepo permission.Repository) *RoleHandler {
	return &RoleHandler{roleRepo: roleRepo, permRepo: permRepo}
}

// GET /api/v1/roles
func (h *RoleHandler) List(c *gin.Context) {
	roles, err := h.roleRepo.List(c.Request.Context())
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, toRoleListResponse(roles))
}

// GET /api/v1/roles/:id
func (h *RoleHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid role id")
		return
	}
	r, err := h.roleRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			response.NotFound(c, "role not found")
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, toRoleResponse(r))
}

// POST /api/v1/roles
func (h *RoleHandler) Create(c *gin.Context) {
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	r := &role.Role{ID: uuid.New(), Name: body.Name}
	if err := h.roleRepo.Create(c.Request.Context(), r); err != nil {
		if errors.Is(err, postgres.ErrDuplicate) {
			response.Conflict(c, "role already exists")
			return
		}
		response.InternalError(c)
		return
	}
	response.Created(c, toRoleResponse(r))
}

// DELETE /api/v1/roles/:id
func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid role id")
		return
	}
	if err := h.roleRepo.Delete(c.Request.Context(), id); err != nil {
		response.InternalError(c)
		return
	}
	response.NoContent(c)
}

// GET /api/v1/roles/permissions
func (h *RoleHandler) ListPermissions(c *gin.Context) {
	perms, err := h.permRepo.List(c.Request.Context())
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, toPermListResponse(perms))
}

// POST /api/v1/roles/:id/permissions
func (h *RoleHandler) AssignPermission(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid role id")
		return
	}
	var body struct {
		Permission string `json:"permission" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	p, err := h.permRepo.FindByName(c.Request.Context(), body.Permission)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			response.NotFound(c, "permission not found")
			return
		}
		response.InternalError(c)
		return
	}
	if err := h.roleRepo.AssignPermission(c.Request.Context(), roleID, p.ID); err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, gin.H{"message": "permission assigned"})
}

// DELETE /api/v1/roles/:id/permissions/:permission
func (h *RoleHandler) RemovePermission(c *gin.Context) {
	roleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid role id")
		return
	}
	p, err := h.permRepo.FindByName(c.Request.Context(), c.Param("permission"))
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			response.NotFound(c, "permission not found")
			return
		}
		response.InternalError(c)
		return
	}
	if err := h.roleRepo.RemovePermission(c.Request.Context(), roleID, p.ID); err != nil {
		response.InternalError(c)
		return
	}
	response.NoContent(c)
}

// ── response shapes ───────────────────────────────────────────────────────────

type roleResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type permResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func toRoleResponse(r *role.Role) roleResponse {
	return roleResponse{ID: r.ID, Name: r.Name}
}

func toRoleListResponse(roles []*role.Role) []roleResponse {
	out := make([]roleResponse, len(roles))
	for i, r := range roles {
		out[i] = toRoleResponse(r)
	}
	return out
}

func toPermListResponse(perms []*permission.Permission) []permResponse {
	out := make([]permResponse, len(perms))
	for i, p := range perms {
		out[i] = permResponse{ID: p.ID, Name: p.Name}
	}
	return out
}
