package entity

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType categorises what triggered the notification.
type NotificationType string

const (
	NotificationTypeJoin  NotificationType = "event_join"
	NotificationTypeLeave NotificationType = "event_leave"
)

// Notification is a push message sent to a user in real-time.
type Notification struct {
	ID        uuid.UUID        `db:"id"         json:"id"`
	UserID    uuid.UUID        `db:"user_id"    json:"user_id"`   // recipient
	EventID   uuid.UUID        `db:"event_id"   json:"event_id"`
	Type      NotificationType `db:"type"       json:"type"`
	Message   string           `db:"message"    json:"message"`
	IsRead    bool             `db:"is_read"    json:"is_read"`
	CreatedAt time.Time        `db:"created_at" json:"created_at"`
}
