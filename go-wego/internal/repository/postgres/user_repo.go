package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	userUC "github.com/go-wego/wego/internal/usecase/user"
	"github.com/go-wego/wego/internal/entity"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// userRepo is the PostgreSQL implementation of user.Repository.
type userRepo struct {
	db *sqlx.DB
}

// NewUserRepo constructs a userRepo.
// It satisfies the user.Repository interface defined in the use-case package.
func NewUserRepo(db *sqlx.DB) userUC.Repository {
	return &userRepo{db: db}
}

// ——— Create ——————————————————————————————————————————————————————————————————

func (r *userRepo) Create(ctx context.Context, u *entity.User) error {
	query := `
		INSERT INTO users (id, username, email, password, role, avatar_url, created_at, updated_at)
		VALUES (:id, :username, :email, :password, :role, :avatar_url, :created_at, :updated_at)
	`
	// Use a context with a DB timeout to prevent slow queries from blocking
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.NamedExecContext(ctx, query, u)
	if err != nil {
		return fmt.Errorf("userRepo.Create: %w", err)
	}
	return nil
}

// ——— FindByID ————————————————————————————————————————————————————————————————

func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var u entity.User
	query := `SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL`
	if err := r.db.GetContext(ctx, &u, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, userUC.ErrNotFound
		}
		return nil, fmt.Errorf("userRepo.FindByID: %w", err)
	}
	return &u, nil
}

// ——— FindByEmail —————————————————————————————————————————————————————————————

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var u entity.User
	query := `SELECT * FROM users WHERE email = $1 AND deleted_at IS NULL`
	if err := r.db.GetContext(ctx, &u, query, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, userUC.ErrNotFound
		}
		return nil, fmt.Errorf("userRepo.FindByEmail: %w", err)
	}
	return &u, nil
}

// ——— Update ——————————————————————————————————————————————————————————————————

func (r *userRepo) Update(ctx context.Context, u *entity.User) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE users
		SET username = :username, avatar_url = :avatar_url, updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL
	`
	_, err := r.db.NamedExecContext(ctx, query, u)
	if err != nil {
		return fmt.Errorf("userRepo.Update: %w", err)
	}
	return nil
}

// ——— SoftDelete ——————————————————————————————————————————————————————————————

func (r *userRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `UPDATE users SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("userRepo.SoftDelete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return userUC.ErrNotFound
	}
	return nil
}
