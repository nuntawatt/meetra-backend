package event_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-wego/wego/internal/entity"
	eventUC "github.com/go-wego/wego/internal/usecase/event"
	notifUC "github.com/go-wego/wego/internal/usecase/notification"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ——— Mocks ————————————————————————————————————————————————————————————————

type mockEventRepo struct{ mock.Mock }

func (m *mockEventRepo) Create(ctx context.Context, e *entity.Event) error {
	return m.Called(ctx, e).Error(0)
}
func (m *mockEventRepo) FindByID(ctx context.Context, id uuid.UUID) (*entity.Event, error) {
	args := m.Called(ctx, id)
	if v, ok := args.Get(0).(*entity.Event); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockEventRepo) List(ctx context.Context, f entity.EventFilter, pg entity.Pagination) ([]*entity.Event, int, error) {
	args := m.Called(ctx, f, pg)
	return args.Get(0).([]*entity.Event), args.Int(1), args.Error(2)
}
func (m *mockEventRepo) Update(ctx context.Context, e *entity.Event) error {
	return m.Called(ctx, e).Error(0)
}
func (m *mockEventRepo) AddParticipant(ctx context.Context, eventID, userID uuid.UUID) error {
	return m.Called(ctx, eventID, userID).Error(0)
}
func (m *mockEventRepo) RemoveParticipant(ctx context.Context, eventID, userID uuid.UUID) error {
	return m.Called(ctx, eventID, userID).Error(0)
}
func (m *mockEventRepo) IsParticipant(ctx context.Context, eventID, userID uuid.UUID) (bool, error) {
	args := m.Called(ctx, eventID, userID)
	return args.Bool(0), args.Error(1)
}
func (m *mockEventRepo) ParticipantCount(ctx context.Context, eventID uuid.UUID) (int, error) {
	args := m.Called(ctx, eventID)
	return args.Int(0), args.Error(1)
}

type mockCache struct{ mock.Mock }

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}
func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl int) error {
	return m.Called(ctx, key, value, ttl).Error(0)
}
func (m *mockCache) Delete(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}

type mockNotifUC struct{ mock.Mock }

func (m *mockNotifUC) NotifyJoin(ctx context.Context, hostID, eventID, joinerID uuid.UUID) error {
	return m.Called(ctx, hostID, eventID, joinerID).Error(0)
}
func (m *mockNotifUC) ListForUser(ctx context.Context, userID uuid.UUID) ([]*entity.Notification, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*entity.Notification), args.Error(1)
}
func (m *mockNotifUC) MarkRead(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

var _ notifUC.UseCase = &mockNotifUC{}

// ——— Tests ————————————————————————————————————————————————————————————————

func TestCreateEvent(t *testing.T) {
	tests := []struct {
		name    string
		input   eventUC.CreateInput
		wantErr error
	}{
		{
			name: "valid event",
			input: eventUC.CreateInput{
				Title:       "Go Meetup",
				Description: "Fun Go meetup",
				Location:    "Bangkok",
				MaxCapacity: 50,
				StartsAt:    time.Now().Add(24 * time.Hour),
				EndsAt:      time.Now().Add(27 * time.Hour),
			},
			wantErr: nil,
		},
		{
			name: "ends before starts",
			input: eventUC.CreateInput{
				Title:       "Bad Event",
				Description: "desc",
				Location:    "Bangkok",
				MaxCapacity: 10,
				StartsAt:    time.Now().Add(3 * time.Hour),
				EndsAt:      time.Now().Add(1 * time.Hour),
			},
			wantErr: eventUC.ErrInvalidTimeRange,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockEventRepo{}
			cache := &mockCache{}
			notif := &mockNotifUC{}

			if tc.wantErr == nil {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Event")).Return(nil)
				cache.On("Delete", mock.Anything, "event:list:page:1").Return(nil)
			}

			uc := eventUC.New(repo, cache, notif)
			hostID := uuid.New()
			e, err := uc.Create(context.Background(), hostID, tc.input)

			if tc.wantErr != nil {
				assert.True(t, errors.Is(err, tc.wantErr))
				assert.Nil(t, e)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, hostID, e.HostID)
				assert.Equal(t, tc.input.Title, e.Title)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestJoinEvent(t *testing.T) {
	eventID := uuid.New()
	hostID := uuid.New()
	userID := uuid.New()

	baseEvent := &entity.Event{
		ID:          eventID,
		HostID:      hostID,
		Status:      entity.EventStatusPublished,
		MaxCapacity: 10,
	}

	tests := []struct {
		name    string
		setup   func(repo *mockEventRepo)
		wantErr error
	}{
		{
			name: "success",
			setup: func(repo *mockEventRepo) {
				repo.On("FindByID", mock.Anything, eventID).Return(baseEvent, nil)
				repo.On("ParticipantCount", mock.Anything, eventID).Return(5, nil)
				repo.On("IsParticipant", mock.Anything, eventID, userID).Return(false, nil)
				repo.On("AddParticipant", mock.Anything, eventID, userID).Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "event full",
			setup: func(repo *mockEventRepo) {
				repo.On("FindByID", mock.Anything, eventID).Return(baseEvent, nil)
				repo.On("ParticipantCount", mock.Anything, eventID).Return(10, nil)
			},
			wantErr: eventUC.ErrEventFull,
		},
		{
			name: "already joined",
			setup: func(repo *mockEventRepo) {
				repo.On("FindByID", mock.Anything, eventID).Return(baseEvent, nil)
				repo.On("ParticipantCount", mock.Anything, eventID).Return(3, nil)
				repo.On("IsParticipant", mock.Anything, eventID, userID).Return(true, nil)
			},
			wantErr: eventUC.ErrAlreadyJoined,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockEventRepo{}
			cache := &mockCache{}
			notif := &mockNotifUC{}

			if tc.wantErr == nil {
				notif.On("NotifyJoin", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil).Maybe()
			}

			tc.setup(repo)
			uc := eventUC.New(repo, cache, notif)
			err := uc.Join(context.Background(), eventID, userID)

			if tc.wantErr != nil {
				assert.True(t, errors.Is(err, tc.wantErr))
			} else {
				assert.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}
