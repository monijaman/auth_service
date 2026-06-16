package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/monir/auth_service/internal/domain/permission"
	"github.com/monir/auth_service/internal/domain/role"
)

type RoleRepo struct {
	db *pgxpool.Pool
}

func NewRoleRepo(db *pgxpool.Pool) *RoleRepo {
	return &RoleRepo{db: db}
}

func (r *RoleRepo) Create(ctx context.Context, ro *role.Role) error {
	_, err := r.db.Exec(ctx, `INSERT INTO roles(id, name) VALUES($1,$2)`, ro.ID, ro.Name)
	if err != nil && isDuplicate(err) {
		return ErrDuplicate
	}
	return err
}

func (r *RoleRepo) FindByID(ctx context.Context, id uuid.UUID) (*role.Role, error) {
	ro := &role.Role{}
	err := r.db.QueryRow(ctx, `SELECT id, name FROM roles WHERE id=$1`, id).Scan(&ro.ID, &ro.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return ro, err
}

func (r *RoleRepo) FindByName(ctx context.Context, name string) (*role.Role, error) {
	ro := &role.Role{}
	err := r.db.QueryRow(ctx, `SELECT id, name FROM roles WHERE name=$1`, name).Scan(&ro.ID, &ro.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return ro, err
}

func (r *RoleRepo) List(ctx context.Context) ([]*role.Role, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name FROM roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*role.Role
	for rows.Next() {
		ro := &role.Role{}
		if err := rows.Scan(&ro.ID, &ro.Name); err != nil {
			return nil, err
		}
		list = append(list, ro)
	}
	return list, rows.Err()
}

func (r *RoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM roles WHERE id=$1`, id)
	return err
}

func (r *RoleRepo) AssignPermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO role_permissions(role_id, permission_id) VALUES($1,$2) ON CONFLICT DO NOTHING`,
		roleID, permissionID,
	)
	return err
}

func (r *RoleRepo) RemovePermission(ctx context.Context, roleID, permissionID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM role_permissions WHERE role_id=$1 AND permission_id=$2`, roleID, permissionID,
	)
	return err
}

// PermissionRepo

type PermissionRepo struct {
	db *pgxpool.Pool
}

func NewPermissionRepo(db *pgxpool.Pool) *PermissionRepo {
	return &PermissionRepo{db: db}
}

func (r *PermissionRepo) Create(ctx context.Context, p *permission.Permission) error {
	_, err := r.db.Exec(ctx, `INSERT INTO permissions(id, name) VALUES($1,$2)`, p.ID, p.Name)
	if err != nil && isDuplicate(err) {
		return ErrDuplicate
	}
	return err
}

func (r *PermissionRepo) FindByID(ctx context.Context, id uuid.UUID) (*permission.Permission, error) {
	p := &permission.Permission{}
	err := r.db.QueryRow(ctx, `SELECT id, name FROM permissions WHERE id=$1`, id).Scan(&p.ID, &p.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *PermissionRepo) FindByName(ctx context.Context, name string) (*permission.Permission, error) {
	p := &permission.Permission{}
	err := r.db.QueryRow(ctx, `SELECT id, name FROM permissions WHERE name=$1`, name).Scan(&p.ID, &p.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *PermissionRepo) List(ctx context.Context) ([]*permission.Permission, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name FROM permissions ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*permission.Permission
	for rows.Next() {
		p := &permission.Permission{}
		if err := rows.Scan(&p.ID, &p.Name); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

func (r *PermissionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM permissions WHERE id=$1`, id)
	return err
}
