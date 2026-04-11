// Package event contains the use-case layer for event management.
package event

import (
	"context"

	"github.com/go-wego/wego/internal/entity"
	"github.com/google/uuid"
)

// Repository defines data-access operations the event use-case needs.
type Repository interface {
	// Create persists a new event.
	Create(ctx context.Context, event *entity.Event) error

	// FindByID retrieves an event by UUID, including host info and participant count.
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Event, error)

	// List returns a paginated, filtered list of events.
	List(ctx context.Context, filter entity.EventFilter, pagination entity.Pagination) ([]*entity.Event, int, error)

	// SoftDelete marks an event as deleted (sets deleted_at = now()).
	// After this, the event will no longer appear in List or FindByID queries.
	SoftDelete(ctx context.Context, id uuid.UUID) error

	// Update updates mutable event fields.
	Update(ctx context.Context, event *entity.Event) error

	// AddParticipant adds a user to an event (idempotent).
	AddParticipant(ctx context.Context, eventID, userID uuid.UUID) error

	// RemoveParticipant removes a user from an event.
	RemoveParticipant(ctx context.Context, eventID, userID uuid.UUID) error

	// IsParticipant checks whether a user has already joined the event.
	IsParticipant(ctx context.Context, eventID, userID uuid.UUID) (bool, error)

	// ParticipantCount returns the current number of participants.
	ParticipantCount(ctx context.Context, eventID uuid.UUID) (int, error)
}

// CacheRepository defines cache operations needed by the event use-case.
type CacheRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error
	Delete(ctx context.Context, key string) error
}
