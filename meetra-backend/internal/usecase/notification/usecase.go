package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/go-wego/wego/internal/entity"
	"github.com/google/uuid"
)

// ——— UseCase —————————————————————————————————————————————————————————————————

// UseCase defines notification business logic.
type UseCase interface {
	// NotifyJoin creates a notification and pushes it to the host via WebSocket.
	NotifyJoin(ctx context.Context, hostID, eventID, joinerID uuid.UUID) error

	// ListForUser returns all notifications for the given user.
	ListForUser(ctx context.Context, userID uuid.UUID) ([]*entity.Notification, error)

	// MarkRead marks a notification as read.
	MarkRead(ctx context.Context, notifID uuid.UUID) error
}

// notificationUseCase is the concrete implementation.
type notificationUseCase struct {
	repo      Repository
	publisher Publisher
}

// New builds a notificationUseCase.
func New(repo Repository, publisher Publisher) UseCase {
	return &notificationUseCase{repo: repo, publisher: publisher}
}

// ——— NotifyJoin ——————————————————————————————————————————————————————————————

// NotifyJoin persists a join notification and fans it out to connected WebSocket clients.
func (uc *notificationUseCase) NotifyJoin(
	ctx context.Context,
	hostID, eventID, joinerID uuid.UUID,
) error {
	n := &entity.Notification{
		ID:        uuid.New(),
		UserID:    hostID, // notification recipient = event host
		EventID:   eventID,
		Type:      entity.NotificationTypeJoin,
		Message:   fmt.Sprintf("A new user has joined your event."),
		IsRead:    false,
		CreatedAt: time.Now().UTC(),
	}

	if err := uc.repo.Create(ctx, n); err != nil {
		return fmt.Errorf("notification.NotifyJoin Create: %w", err)
	}

	// Non-blocking push to connected WebSocket clients
	uc.publisher.Publish(hostID, n)

	return nil
}

// ——— ListForUser —————————————————————————————————————————————————————————————

func (uc *notificationUseCase) ListForUser(ctx context.Context, userID uuid.UUID) ([]*entity.Notification, error) {
	return uc.repo.ListByUser(ctx, userID)
}

// ——— MarkRead ————————————————————————————————————————————————————————————————

func (uc *notificationUseCase) MarkRead(ctx context.Context, notifID uuid.UUID) error {
	return uc.repo.MarkRead(ctx, notifID)
}
