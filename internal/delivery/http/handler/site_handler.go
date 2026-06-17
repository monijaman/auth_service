package handler

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/monir/auth_service/internal/domain/site"
	"github.com/monir/auth_service/internal/domain/role"
	"github.com/monir/auth_service/internal/repository/postgres"
	"github.com/monir/auth_service/pkg/response"
)

type SiteHandler struct {
	siteRepo site.Repository
	roleRepo role.Repository
}

func NewSiteHandler(siteRepo site.Repository, roleRepo role.Repository) *SiteHandler {
	return &SiteHandler{siteRepo: siteRepo, roleRepo: roleRepo}
}

// GET /api/v1/sites
func (h *SiteHandler) List(c *gin.Context) {
	sites, err := h.siteRepo.List(c.Request.Context())
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, toSiteListResponse(sites))
}

// GET /api/v1/sites/:id
func (h *SiteHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid site id")
		return
	}
	s, err := h.siteRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			response.NotFound(c, "site not found")
			return
		}
		response.InternalError(c)
		return
	}
	response.OK(c, toSiteResponse(s))
}

// POST /api/v1/sites
func (h *SiteHandler) Create(c *gin.Context) {
	var body struct {
		Name   string `json:"name"   binding:"required"`
		Slug   string `json:"slug"   binding:"required"`
		Domain string `json:"domain"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	s := &site.Site{
		ID:        uuid.New(),
		Name:      body.Name,
		Slug:      body.Slug,
		Domain:    body.Domain,
		IsActive:  true,
		CreatedAt: time.Now(),
	}
	if err := h.siteRepo.Create(c.Request.Context(), s); err != nil {
		if errors.Is(err, postgres.ErrDuplicate) {
			response.Conflict(c, "site name or slug already exists")
			return
		}
		response.InternalError(c)
		return
	}
	response.Created(c, toSiteResponse(s))
}

// PUT /api/v1/sites/:id
func (h *SiteHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid site id")
		return
	}
	s, err := h.siteRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			response.NotFound(c, "site not found")
			return
		}
		response.InternalError(c)
		return
	}
	var body struct {
		Name     string `json:"name"`
		Slug     string `json:"slug"`
		Domain   string `json:"domain"`
		IsActive *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if body.Name != "" {
		s.Name = body.Name
	}
	if body.Slug != "" {
		s.Slug = body.Slug
	}
	if body.Domain != "" {
		s.Domain = body.Domain
	}
	if body.IsActive != nil {
		s.IsActive = *body.IsActive
	}
	if err := h.siteRepo.Update(c.Request.Context(), s); err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, toSiteResponse(s))
}

// DELETE /api/v1/sites/:id
func (h *SiteHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid site id")
		return
	}
	if err := h.siteRepo.Delete(c.Request.Context(), id); err != nil {
		response.InternalError(c)
		return
	}
	response.NoContent(c)
}

// GET /api/v1/sites/:id/users
func (h *SiteHandler) ListUsers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid site id")
		return
	}
	users, err := h.siteRepo.GetSiteUsers(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, users)
}

// POST /api/v1/sites/:id/users/:user_id/roles
func (h *SiteHandler) AssignUserRole(c *gin.Context) {
	siteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid site id")
		return
	}
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	var body struct {
		RoleName string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	r, err := h.roleRepo.FindByName(c.Request.Context(), body.RoleName)
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			response.NotFound(c, "role not found")
			return
		}
		response.InternalError(c)
		return
	}
	if err := h.siteRepo.AssignUserRole(c.Request.Context(), userID, siteID, r.ID); err != nil {
		response.InternalError(c)
		return
	}
	response.OK(c, gin.H{"message": "role assigned"})
}

// DELETE /api/v1/sites/:id/users/:user_id/roles/:role
func (h *SiteHandler) RemoveUserRole(c *gin.Context) {
	siteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid site id")
		return
	}
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	r, err := h.roleRepo.FindByName(c.Request.Context(), c.Param("role"))
	if err != nil {
		if errors.Is(err, postgres.ErrNotFound) {
			response.NotFound(c, "role not found")
			return
		}
		response.InternalError(c)
		return
	}
	if err := h.siteRepo.RemoveUserRole(c.Request.Context(), userID, siteID, r.ID); err != nil {
		response.InternalError(c)
		return
	}
	response.NoContent(c)
}

// ── response shapes ───────────────────────────────────────────────────────────

type siteResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Slug     string    `json:"slug"`
	Domain   string    `json:"domain"`
	IsActive bool      `json:"is_active"`
}

func toSiteResponse(s *site.Site) siteResponse {
	return siteResponse{ID: s.ID, Name: s.Name, Slug: s.Slug, Domain: s.Domain, IsActive: s.IsActive}
}

func toSiteListResponse(sites []*site.Site) []siteResponse {
	out := make([]siteResponse, len(sites))
	for i, s := range sites {
		out[i] = toSiteResponse(s)
	}
	return out
}
