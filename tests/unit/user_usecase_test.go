package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nuntawatt/meetra-backend/internal/auth"
	"github.com/nuntawatt/meetra-backend/internal/config"
	"github.com/nuntawatt/meetra-backend/internal/domain"
	userUC "github.com/nuntawatt/meetra-backend/internal/usecase/user"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ——— Mock implementations ——————————————————————————————————————————————————

// mockUserRepo implements user.Repository using testify/mock.
type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Create(ctx context.Context, u *domain.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if v, ok := args.Get(0).(*domain.User); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	if v, ok := args.Get(0).(*domain.User); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}
func (m *mockUserRepo) Update(ctx context.Context, u *domain.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *mockUserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// mockCacheRepo implements user.CacheRepository.
type mockCacheRepo struct{ mock.Mock }

func (m *mockCacheRepo) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}
func (m *mockCacheRepo) Set(ctx context.Context, key string, value interface{}, ttl int) error {
	return m.Called(ctx, key, value, ttl).Error(0)
}
func (m *mockCacheRepo) Delete(ctx context.Context, key string) error {
	return m.Called(ctx, key).Error(0)
}

// ——— Helper ————————————————————————————————————————————————————————————————

func newTestUseCase(repo *mockUserRepo, cache *mockCacheRepo) userUC.UseCase {
	jwtSvc := auth.NewJWTService("test-access", "test-refresh")
	bcryptSvc := auth.NewBcryptService(4) // low cost for fast tests
	authCfg := config.AuthConfig{
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	}
	return userUC.New(repo, cache, authCfg, jwtSvc, bcryptSvc)
}

// ——— Table-driven tests ———————————————————————————————————————————————————

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		input   userUC.RegisterInput
		setup   func(repo *mockUserRepo)
		wantErr error
	}{
		{
			name: "success",
			input: userUC.RegisterInput{
				Username: "testuser",
				Email:    "test@example.com",
				Password: "password123",
			},
			setup: func(repo *mockUserRepo) {
				// Email does not exist
				repo.On("FindByEmail", mock.Anything, "test@example.com").
					Return(nil, userUC.ErrNotFound)
				// Create succeeds
				repo.On("Create", mock.Anything, mock.AnythingOfType("*domain.User")).
					Return(nil)
			},
			wantErr: nil,
		},
		{
			name: "email already taken",
			input: userUC.RegisterInput{
				Username: "dup",
				Email:    "existing@example.com",
				Password: "password123",
			},
			setup: func(repo *mockUserRepo) {
				existing := &domain.User{ID: uuid.New(), Email: "existing@example.com"}
				repo.On("FindByEmail", mock.Anything, "existing@example.com").
					Return(existing, nil)
			},
			wantErr: userUC.ErrEmailTaken,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockUserRepo{}
			cache := &mockCacheRepo{}
			tc.setup(repo)

			uc := newTestUseCase(repo, cache)
			u, err := uc.Register(context.Background(), tc.input)

			if tc.wantErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tc.wantErr))
				assert.Nil(t, u)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, u)
				assert.Equal(t, tc.input.Email, u.Email)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestLogin(t *testing.T) {
	// Pre-hashed password for "password123" with bcrypt cost 4
	bcryptSvc := auth.NewBcryptService(4)
	hashed, _ := bcryptSvc.Hash("password123")

	tests := []struct {
		name    string
		input   userUC.LoginInput
		setup   func(repo *mockUserRepo)
		wantErr error
	}{
		{
			name:  "valid credentials",
			input: userUC.LoginInput{Email: "user@example.com", Password: "password123"},
			setup: func(repo *mockUserRepo) {
				repo.On("FindByEmail", mock.Anything, "user@example.com").Return(&domain.User{
					ID:       uuid.New(),
					Email:    "user@example.com",
					Password: hashed,
					Role:     domain.RoleUser,
				}, nil)
			},
			wantErr: nil,
		},
		{
			name:  "wrong password",
			input: userUC.LoginInput{Email: "user@example.com", Password: "wrongpass"},
			setup: func(repo *mockUserRepo) {
				repo.On("FindByEmail", mock.Anything, "user@example.com").Return(&domain.User{
					ID:       uuid.New(),
					Email:    "user@example.com",
					Password: hashed,
					Role:     domain.RoleUser,
				}, nil)
			},
			wantErr: userUC.ErrInvalidCredentials,
		},
		{
			name:  "user not found",
			input: userUC.LoginInput{Email: "nobody@example.com", Password: "pass"},
			setup: func(repo *mockUserRepo) {
				repo.On("FindByEmail", mock.Anything, "nobody@example.com").
					Return(nil, userUC.ErrNotFound)
			},
			wantErr: userUC.ErrInvalidCredentials,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockUserRepo{}
			cache := &mockCacheRepo{}
			tc.setup(repo)

			uc := newTestUseCase(repo, cache)
			resp, err := uc.Login(context.Background(), tc.input)

			if tc.wantErr != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tc.wantErr))
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)
			}
			repo.AssertExpectations(t)
		})
	}
}
