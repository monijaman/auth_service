package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/monir/auth_service/internal/domain/site"
)

type SiteRepo struct {
	db *pgxpool.Pool
}

func NewSiteRepo(db *pgxpool.Pool) *SiteRepo {
	return &SiteRepo{db: db}
}

func (r *SiteRepo) FindByID(ctx context.Context, id uuid.UUID) (*site.Site, error) {
	s := &site.Site{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, slug, COALESCE(domain,''), is_active, created_at FROM sites WHERE id=$1`, id,
	).Scan(&s.ID, &s.Name, &s.Slug, &s.Domain, &s.IsActive, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *SiteRepo) FindBySlug(ctx context.Context, slug string) (*site.Site, error) {
	s := &site.Site{}
	err := r.db.QueryRow(ctx,
		`SELECT id, name, slug, COALESCE(domain,''), is_active, created_at FROM sites WHERE slug=$1`, slug,
	).Scan(&s.ID, &s.Name, &s.Slug, &s.Domain, &s.IsActive, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *SiteRepo) Create(ctx context.Context, s *site.Site) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO sites(id, name, slug, domain, is_active, created_at) VALUES($1,$2,$3,NULLIF($4,''),$5,$6)`,
		s.ID, s.Name, s.Slug, s.Domain, s.IsActive, s.CreatedAt,
	)
	if err != nil && isDuplicate(err) {
		return ErrDuplicate
	}
	return err
}

func (r *SiteRepo) List(ctx context.Context) ([]*site.Site, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, slug, COALESCE(domain,''), is_active, created_at FROM sites ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*site.Site
	for rows.Next() {
		s := &site.Site{}
		if err := rows.Scan(&s.ID, &s.Name, &s.Slug, &s.Domain, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

func (r *SiteRepo) Update(ctx context.Context, s *site.Site) error {
	_, err := r.db.Exec(ctx,
		`UPDATE sites SET name=$1, slug=$2, domain=NULLIF($3,''), is_active=$4 WHERE id=$5`,
		s.Name, s.Slug, s.Domain, s.IsActive, s.ID,
	)
	return err
}

func (r *SiteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sites WHERE id=$1`, id)
	return err
}

func (r *SiteRepo) RemoveUserRole(ctx context.Context, userID, siteID, roleID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM user_site_roles WHERE user_id=$1 AND site_id=$2 AND role_id=$3`,
		userID, siteID, roleID,
	)
	return err
}

func (r *SiteRepo) GetSiteUsers(ctx context.Context, siteID uuid.UUID) ([]*site.SiteUser, error) {
	rows, err := r.db.Query(ctx,
		`SELECT u.id, u.email, array_agg(ro.name) AS roles
		 FROM users u
		 JOIN user_site_roles usr ON u.id = usr.user_id
		 JOIN roles ro ON ro.id = usr.role_id
		 WHERE usr.site_id=$1 AND u.deleted_at IS NULL
		 GROUP BY u.id, u.email`, siteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*site.SiteUser
	for rows.Next() {
		su := &site.SiteUser{}
		if err := rows.Scan(&su.UserID, &su.Email, &su.Roles); err != nil {
			return nil, err
		}
		users = append(users, su)
	}
	return users, rows.Err()
}

func (r *SiteRepo) AssignUserRole(ctx context.Context, userID, siteID, roleID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO user_site_roles(user_id, site_id, role_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`,
		userID, siteID, roleID,
	)
	return err
}

func (r *SiteRepo) GetUserRoles(ctx context.Context, userID, siteID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT ro.name FROM roles ro
		 JOIN user_site_roles usr ON ro.id = usr.role_id
		 WHERE usr.user_id=$1 AND usr.site_id=$2`, userID, siteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		roles = append(roles, name)
	}
	return roles, rows.Err()
}

func (r *SiteRepo) GetUserPermissions(ctx context.Context, userID, siteID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT p.name FROM permissions p
		 JOIN role_permissions rp  ON p.id  = rp.permission_id
		 JOIN user_site_roles  usr ON rp.role_id = usr.role_id
		 WHERE usr.user_id=$1 AND usr.site_id=$2`, userID, siteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var perms []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		perms = append(perms, name)
	}
	return perms, rows.Err()
}

func (r *SiteRepo) GetUserSites(ctx context.Context, userID uuid.UUID) ([]*site.Site, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT s.id, s.name, s.slug, COALESCE(s.domain,''), s.is_active, s.created_at
		 FROM sites s
		 JOIN user_site_roles usr ON s.id = usr.site_id
		 WHERE usr.user_id=$1 AND s.is_active=true`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sites []*site.Site
	for rows.Next() {
		s := &site.Site{}
		if err := rows.Scan(&s.ID, &s.Name, &s.Slug, &s.Domain, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		sites = append(sites, s)
	}
	return sites, rows.Err()
}
