package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptService wraps password hashing with bcrypt.
type BcryptService struct {
	cost int
}

// NewBcryptService creates a BcryptService with the given cost factor.
// bcrypt.DefaultCost (10) is appropriate for production.
func NewBcryptService(cost int) *BcryptService {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &BcryptService{cost: cost}
}

// Hash generates a bcrypt hash of the plaintext password.
func (s *BcryptService) Hash(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt.Hash: %w", err)
	}
	return string(hashed), nil
}

// Compare verifies a plaintext password against a stored hash.
// Returns nil if they match, an error otherwise.
func (s *BcryptService) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
