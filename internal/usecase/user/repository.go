// Package user contains the use-case layer for user management.
// Repository interfaces are defined HERE (consumer owns the contract).
// The concrete implementations live in internal/repository/postgres/.
package user

import (
	"context"

	"github.com/nuntawatt/meetra-backend/internal/domain"
	"github.com/google/uuid"
)

// Repository defines the data-access operations the user use-case needs.
// By placing the interface in this package, the use-case is the client that
// dictates the contract — concrete implementations must satisfy it.
type Repository interface {
	// Create persists a new user and returns the created record.
	Create(ctx context.Context, user *domain.User) error

	// FindByID retrieves a user by their UUID.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	// FindByEmail retrieves a user by their email address.
	FindByEmail(ctx context.Context, email string) (*domain.User, error)

	// Update updates mutable user fields (username, avatar_url).
	Update(ctx context.Context, user *domain.User) error

	// SoftDelete marks deleted_at so the row is logically deleted.
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// CacheRepository defines cache operations needed by the user use-case.
type CacheRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error
	Delete(ctx context.Context, key string) error
}
