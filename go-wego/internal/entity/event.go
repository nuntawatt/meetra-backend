package entity

import (
	"time"

	"github.com/google/uuid"
)

// EventStatus represents the lifecycle state of an event.
type EventStatus string

const (
	EventStatusDraft     EventStatus = "draft"
	EventStatusPublished EventStatus = "published"
	EventStatusCancelled EventStatus = "cancelled"
	EventStatusCompleted EventStatus = "completed"
)

// Event is the core event domain model.
type Event struct {
	ID          uuid.UUID   `db:"id"          json:"id"`
	HostID      uuid.UUID   `db:"host_id"     json:"host_id"`
	Title       string      `db:"title"       json:"title"`
	Description string      `db:"description" json:"description"`
	Location    string      `db:"location"    json:"location"`
	ImageURL    string      `db:"image_url"   json:"image_url,omitempty"`
	MaxCapacity int         `db:"max_capacity" json:"max_capacity"`
	Status      EventStatus `db:"status"      json:"status"`
	StartsAt    time.Time   `db:"starts_at"   json:"starts_at"`
	EndsAt      time.Time   `db:"ends_at"     json:"ends_at"`
	CreatedAt   time.Time   `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time   `db:"updated_at"  json:"updated_at"`
	DeletedAt   *time.Time  `db:"deleted_at"  json:"deleted_at,omitempty"` // nil = active; non-nil = soft deleted

	// Computed fields (not stored — populated by JOIN queries)
	HostUsername     string `db:"host_username"    json:"host_username,omitempty"`
	ParticipantCount int    `db:"participant_count" json:"participant_count,omitempty"`
}

// EventParticipant records the many-to-many relationship between users and events.
type EventParticipant struct {
	EventID  uuid.UUID `db:"event_id"  json:"event_id"`
	UserID   uuid.UUID `db:"user_id"   json:"user_id"`
	JoinedAt time.Time `db:"joined_at" json:"joined_at"`
}

// EventFilter contains optional filters for listing events.
type EventFilter struct {
	Status   EventStatus
	Location string
	Search   string // full-text search on title/description
}

// Pagination holds limit/offset for list queries.
type Pagination struct {
	Page  int // 1-based
	Limit int
}

// Offset calculates the SQL OFFSET value.
func (p Pagination) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit
}
