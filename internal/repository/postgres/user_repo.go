package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/monir/auth_service/internal/domain/user"
)

var ErrNotFound = errors.New("record not found")
var ErrDuplicate = errors.New("duplicate record")

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u *user.User) error {
	q := `INSERT INTO users (id, email, phone, password_hash, email_verified, status, created_at, updated_at)
	      VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := r.db.Exec(ctx, q,
		u.ID, u.Email, u.Phone, u.PasswordHash,
		u.EmailVerified, u.Status, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if isDuplicate(err) {
			return ErrDuplicate
		}
		return err
	}
	return nil
}

func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	q := `SELECT id, email, phone, password_hash, email_verified, status, deleted_at, created_at, updated_at
	      FROM users WHERE id=$1 AND deleted_at IS NULL`
	return r.scan(r.db.QueryRow(ctx, q, id))
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	q := `SELECT id, email, phone, password_hash, email_verified, status, deleted_at, created_at, updated_at
	      FROM users WHERE email=$1 AND deleted_at IS NULL`
	return r.scan(r.db.QueryRow(ctx, q, email))
}

func (r *UserRepo) Update(ctx context.Context, u *user.User) error {
	q := `UPDATE users SET email=$1, phone=$2, email_verified=$3, status=$4, updated_at=$5 WHERE id=$6`
	u.UpdatedAt = time.Now()
	_, err := r.db.Exec(ctx, q, u.Email, u.Phone, u.EmailVerified, u.Status, u.UpdatedAt, u.ID)
	return err
}

func (r *UserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET deleted_at=$1 WHERE id=$2`, time.Now(), id)
	return err
}

func (r *UserRepo) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO user_roles(user_id, role_id) VALUES($1,$2) ON CONFLICT DO NOTHING`,
		userID, roleID,
	)
	return err
}

func (r *UserRepo) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM user_roles WHERE user_id=$1 AND role_id=$2`, userID, roleID)
	return err
}

func (r *UserRepo) GetRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT r.name FROM roles r
		 JOIN user_roles ur ON r.id=ur.role_id
		 WHERE ur.user_id=$1`, userID,
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

func (r *UserRepo) GetPermissions(ctx context.Context, userID uuid.UUID) ([]string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT p.name FROM permissions p
		 JOIN role_permissions rp ON p.id=rp.permission_id
		 JOIN user_roles ur ON rp.role_id=ur.role_id
		 WHERE ur.user_id=$1`, userID,
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

func (r *UserRepo) scan(row pgx.Row) (*user.User, error) {
	u := &user.User{}
	err := row.Scan(
		&u.ID, &u.Email, &u.Phone, &u.PasswordHash,
		&u.EmailVerified, &u.Status, &u.DeletedAt,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func isDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
