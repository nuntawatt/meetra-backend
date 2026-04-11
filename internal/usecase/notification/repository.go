// Package notification contains the use-case for real-time notifications.
package notification

import (
	"context"

	"github.com/nuntawatt/meetra-backend/internal/domain"
	"github.com/google/uuid"
)

// Repository defines data-access operations the notification use-case needs.
type Repository interface {
	// Create persists a notification record.
	Create(ctx context.Context, n *domain.Notification) error

	// ListByUser returns all notifications for a given user.
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*domain.Notification, error)

	// MarkRead marks a notification as read.
	MarkRead(ctx context.Context, id uuid.UUID) error
}

// Publisher defines the interface for pushing a notification to connected
// WebSocket clients. The WebSocket hub implements this interface.
type Publisher interface {
	Publish(userID uuid.UUID, n *domain.Notification)
}
