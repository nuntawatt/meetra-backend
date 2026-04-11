package postgres

import (
	"context"
	"fmt"
	"time"

	notifUC "github.com/go-wego/wego/internal/usecase/notification"
	"github.com/go-wego/wego/internal/entity"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// notificationRepo is the PostgreSQL implementation of notification.Repository.
type notificationRepo struct {
	db *sqlx.DB
}

// NewNotificationRepo constructs a notificationRepo.
func NewNotificationRepo(db *sqlx.DB) notifUC.Repository {
	return &notificationRepo{db: db}
}

func (r *notificationRepo) Create(ctx context.Context, n *entity.Notification) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO notifications (id, user_id, event_id, type, message, is_read, created_at)
		VALUES (:id, :user_id, :event_id, :type, :message, :is_read, :created_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, n)
	return err
}

func (r *notificationRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]*entity.Notification, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var notifs []*entity.Notification
	err := r.db.SelectContext(ctx, &notifs,
		`SELECT * FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("notifRepo.ListByUser: %w", err)
	}
	return notifs, nil
}

func (r *notificationRepo) MarkRead(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE notifications SET is_read = true WHERE id = $1`,
		id,
	)
	return err
}
