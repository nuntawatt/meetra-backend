package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	eventUC "github.com/nuntawatt/meetra-backend/internal/usecase/event"
	"github.com/nuntawatt/meetra-backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// eventRepo is the PostgreSQL implementation of event.Repository.
type eventRepo struct {
	db *sqlx.DB
}

// NewEventRepo constructs an eventRepo.
func NewEventRepo(db *sqlx.DB) eventUC.Repository {
	return &eventRepo{db: db}
}

// ——— Create ——————————————————————————————————————————————————————————————————

func (r *eventRepo) Create(ctx context.Context, e *domain.Event) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO events
			(id, host_id, title, description, location, image_url, max_capacity, status, starts_at, ends_at, created_at, updated_at)
		VALUES
			(:id, :host_id, :title, :description, :location, :image_url, :max_capacity, :status, :starts_at, :ends_at, :created_at, :updated_at)
	`
	_, err := r.db.NamedExecContext(ctx, query, e)
	if err != nil {
		return fmt.Errorf("eventRepo.Create: %w", err)
	}
	return nil
}

// ——— FindByID ————————————————————————————————————————————————————————————————

func (r *eventRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var e domain.Event
	query := `
		SELECT e.*,
			   u.username AS host_username,
			   COUNT(ep.user_id) AS participant_count
		FROM   events e
		JOIN   users u ON u.id = e.host_id
		LEFT   JOIN event_participants ep ON ep.event_id = e.id
		WHERE  e.id = $1
		  AND  e.deleted_at IS NULL
		GROUP  BY e.id, u.username
	`
	if err := r.db.GetContext(ctx, &e, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, eventUC.ErrNotFound
		}
		return nil, fmt.Errorf("eventRepo.FindByID: %w", err)
	}
	return &e, nil
}

// ——— List ————————————————————————————————————————————————————————————————————

// List returns a paginated list of events with optional filtering.
// It returns the matching events AND the total count (for pagination metadata).
func (r *eventRepo) List(
	ctx context.Context,
	filter domain.EventFilter,
	pg domain.Pagination,
) ([]*domain.Event, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Base condition always excludes soft-deleted events
	conditions := []string{"e.deleted_at IS NULL"}
	args := []interface{}{}
	argIdx := 1

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("e.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}
	if filter.Location != "" {
		conditions = append(conditions, fmt.Sprintf("e.location ILIKE $%d", argIdx))
		args = append(args, "%"+filter.Location+"%")
		argIdx++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(e.title ILIKE $%d OR e.description ILIKE $%d)", argIdx, argIdx+1,
		))
		args = append(args, "%"+filter.Search+"%", "%"+filter.Search+"%")
		argIdx += 2
	}

	where := strings.Join(conditions, " AND ")

	// Count query (no LIMIT/OFFSET)
	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM events e WHERE %s`, where)
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("eventRepo.List count: %w", err)
	}

	// Data query with pagination
	listQuery := fmt.Sprintf(`
		SELECT e.*,
			   u.username AS host_username,
			   COUNT(ep.user_id) AS participant_count
		FROM   events e
		JOIN   users u ON u.id = e.host_id
		LEFT   JOIN event_participants ep ON ep.event_id = e.id
		WHERE  %s
		GROUP  BY e.id, u.username
		ORDER  BY e.starts_at DESC
		LIMIT  $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, pg.Limit, pg.Offset())

	var events []*domain.Event
	if err := r.db.SelectContext(ctx, &events, listQuery, args...); err != nil {
		return nil, 0, fmt.Errorf("eventRepo.List select: %w", err)
	}

	return events, total, nil
}

// ——— Update ——————————————————————————————————————————————————————————————————

func (r *eventRepo) Update(ctx context.Context, e *domain.Event) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	query := `
		UPDATE events
		SET title = :title, description = :description, location = :location,
		    image_url = :image_url, max_capacity = :max_capacity,
		    status = :status, starts_at = :starts_at, ends_at = :ends_at,
		    updated_at = :updated_at
		WHERE id = :id
		  AND deleted_at IS NULL
	`
	_, err := r.db.NamedExecContext(ctx, query, e)
	return err
}

// ——— Participants —————————————————————————————————————————————————————————————

func (r *eventRepo) AddParticipant(ctx context.Context, eventID, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO event_participants (event_id, user_id, joined_at)
		 VALUES ($1, $2, now())
		 ON CONFLICT DO NOTHING`,
		eventID, userID,
	)
	return err
}

func (r *eventRepo) RemoveParticipant(ctx context.Context, eventID, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`DELETE FROM event_participants WHERE event_id = $1 AND user_id = $2`,
		eventID, userID,
	)
	return err
}

func (r *eventRepo) IsParticipant(ctx context.Context, eventID, userID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM event_participants WHERE event_id = $1 AND user_id = $2`,
		eventID, userID,
	)
	return count > 0, err
}

func (r *eventRepo) ParticipantCount(ctx context.Context, eventID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var count int
	err := r.db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM event_participants WHERE event_id = $1`,
		eventID,
	)
	return count, err
}

// ——— SoftDelete —————————————————————————————————————————————————————————————————

// SoftDelete sets deleted_at = now() on the event row.
// The session timezone is Asia/Bangkok so now() returns Thai time.
func (r *eventRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	res, err := r.db.ExecContext(ctx,
		`UPDATE events SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return fmt.Errorf("eventRepo.SoftDelete: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return eventUC.ErrNotFound
	}
	return nil
}
