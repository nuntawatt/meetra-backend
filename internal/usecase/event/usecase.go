package event

import (
	"context"
	"errors"
	"fmt"
	"time"

	notificationUC "github.com/go-wego/wego/internal/usecase/notification"
	"github.com/go-wego/wego/internal/entity"
	"github.com/google/uuid"
)

// bangkokLoc is the Thai timezone used for all time.Now() calls.
// PostgreSQL already stores UTC and converts via the session timezone;
// this location is used for any time value created purely in Go code.
var bangkokLoc = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		return time.UTC // safe fallback
	}
	return loc
}()

// nowBKK returns the current time in Asia/Bangkok (UTC+7).
func nowBKK() time.Time { return time.Now().In(bangkokLoc) }

// ——— DTOs ————————————————————————————————————————————————————————————————————

// CreateInput is the validated payload for creating an event.
type CreateInput struct {
	Title       string      `json:"title"        validate:"required,min=3,max=200"`
	Description string      `json:"description"  validate:"required"`
	Location    string      `json:"location"     validate:"required"`
	MaxCapacity int         `json:"max_capacity" validate:"required,min=1,max=10000"`
	StartsAt    time.Time   `json:"starts_at"    validate:"required"`
	EndsAt      time.Time   `json:"ends_at"      validate:"required"`
}

// ListInput holds pagination and filter params for listing events.
type ListInput struct {
	Page     int
	Limit    int
	Status   string
	Location string
	Search   string
}

// ListResult wraps the list response with total count.
type ListResult struct {
	Events []*entity.Event `json:"events"`
	Total  int             `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}

// ——— UseCase —————————————————————————————————————————————————————————————————

// UseCase defines all event business logic operations.
type UseCase interface {
	Create(ctx context.Context, hostID uuid.UUID, in CreateInput) (*entity.Event, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Event, error)
	List(ctx context.Context, in ListInput) (*ListResult, error)
	Join(ctx context.Context, eventID, userID uuid.UUID) error
	Leave(ctx context.Context, eventID, userID uuid.UUID) error
	Delete(ctx context.Context, eventID, requesterID uuid.UUID) error
}

// eventUseCase is the concrete implementation.
type eventUseCase struct {
	repo        Repository
	cache       CacheRepository
	notifUC     notificationUC.UseCase
}

// New constructs an eventUseCase with injected dependencies.
func New(
	repo Repository,
	cache CacheRepository,
	notifUC notificationUC.UseCase,
) UseCase {
	return &eventUseCase{
		repo:    repo,
		cache:   cache,
		notifUC: notifUC,
	}
}

// ——— Create ——————————————————————————————————————————————————————————————————

// Create validates the time range and persists a new event.
func (uc *eventUseCase) Create(ctx context.Context, hostID uuid.UUID, in CreateInput) (*entity.Event, error) {
	if !in.EndsAt.After(in.StartsAt) {
		return nil, ErrInvalidTimeRange
	}

	now := nowBKK() // Thai time for all created_at / updated_at
	e := &entity.Event{
		ID:          uuid.New(),
		HostID:      hostID,
		Title:       in.Title,
		Description: in.Description,
		Location:    in.Location,
		MaxCapacity: in.MaxCapacity,
		Status:      entity.EventStatusPublished,
		StartsAt:    in.StartsAt,
		EndsAt:      in.EndsAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.repo.Create(ctx, e); err != nil {
		return nil, fmt.Errorf("event.Create: %w", err)
	}

	// Bust list page cache
	_ = uc.cache.Delete(ctx, "event:list:page:1")

	return e, nil
}

// ——— GetByID —————————————————————————————————————————————————————————————————

// GetByID retrieves a single event. Cache-aside pattern: check Redis, then DB.
func (uc *eventUseCase) GetByID(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
	e, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("event.GetByID: %w", err)
	}
	return e, nil
}

// ——— List ————————————————————————————————————————————————————————————————————

// List returns a paginated list of events matching the filter.
func (uc *eventUseCase) List(ctx context.Context, in ListInput) (*ListResult, error) {
	if in.Limit <= 0 {
		in.Limit = 20
	}
	if in.Page <= 0 {
		in.Page = 1
	}

	filter := entity.EventFilter{
		Location: in.Location,
		Search:   in.Search,
	}
	if in.Status != "" {
		filter.Status = entity.EventStatus(in.Status)
	}

	pagination := entity.Pagination{Page: in.Page, Limit: in.Limit}

	events, total, err := uc.repo.List(ctx, filter, pagination)
	if err != nil {
		return nil, fmt.Errorf("event.List: %w", err)
	}

	return &ListResult{
		Events: events,
		Total:  total,
		Page:   in.Page,
		Limit:  in.Limit,
	}, nil
}

// ——— Join ————————————————————————————————————————————————————————————————————

// Join adds a user to an event and triggers a real-time notification to the host.
func (uc *eventUseCase) Join(ctx context.Context, eventID, userID uuid.UUID) error {
	e, err := uc.repo.FindByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("event.Join FindByID: %w", err)
	}

	// Guard: cannot join a cancelled/completed event
	if e.Status == entity.EventStatusCancelled || e.Status == entity.EventStatusCompleted {
		return ErrEventNotJoinable
	}

	// Guard: capacity check
	count, err := uc.repo.ParticipantCount(ctx, eventID)
	if err != nil {
		return fmt.Errorf("event.Join ParticipantCount: %w", err)
	}
	if count >= e.MaxCapacity {
		return ErrEventFull
	}

	// Guard: idempotent check
	already, err := uc.repo.IsParticipant(ctx, eventID, userID)
	if err != nil {
		return fmt.Errorf("event.Join IsParticipant: %w", err)
	}
	if already {
		return ErrAlreadyJoined
	}

	if err := uc.repo.AddParticipant(ctx, eventID, userID); err != nil {
		return fmt.Errorf("event.Join AddParticipant: %w", err)
	}

	// Fire async notification to the event host
	go func() {
		bgCtx := context.Background()
		_ = uc.notifUC.NotifyJoin(bgCtx, e.HostID, eventID, userID)
	}()

	return nil
}

// ——— Leave ———————————————————————————————————————————————————————————————————

// Leave removes a user from an event.
func (uc *eventUseCase) Leave(ctx context.Context, eventID, userID uuid.UUID) error {
	already, err := uc.repo.IsParticipant(ctx, eventID, userID)
	if err != nil {
		return fmt.Errorf("event.Leave IsParticipant: %w", err)
	}
	if !already {
		return ErrNotParticipant
	}

	if err := uc.repo.RemoveParticipant(ctx, eventID, userID); err != nil {
		return fmt.Errorf("event.Leave RemoveParticipant: %w", err)
	}

	return nil
}

// ——— Delete ——————————————————————————————————————————————————————————————————

// Delete soft-deletes an event. Only the host (or admin role) should be allowed;
// that RBAC check happens at the handler/middleware layer.
func (uc *eventUseCase) Delete(ctx context.Context, eventID, requesterID uuid.UUID) error {
	e, err := uc.repo.FindByID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("event.Delete FindByID: %w", err)
	}

	// Only the host may delete (extra safety net alongside middleware)
	if e.HostID != requesterID {
		return ErrForbidden
	}

	if err := uc.repo.SoftDelete(ctx, eventID); err != nil {
		return fmt.Errorf("event.Delete SoftDelete: %w", err)
	}

	// Invalidate any cached data for this event
	_ = uc.cache.Delete(ctx, fmt.Sprintf("event:%s", eventID))
	_ = uc.cache.Delete(ctx, "event:list:page:1")

	return nil
}

// ——— Sentinel errors —————————————————————————————————————————————————————————

var (
	ErrNotFound         = errors.New("event not found")
	ErrInvalidTimeRange = errors.New("ends_at must be after starts_at")
	ErrEventFull        = errors.New("event has reached its capacity")
	ErrEventNotJoinable = errors.New("event is not open for joining")
	ErrAlreadyJoined    = errors.New("user already joined this event")
	ErrNotParticipant   = errors.New("user is not a participant of this event")
	ErrForbidden        = errors.New("only the event host can perform this action")
)
