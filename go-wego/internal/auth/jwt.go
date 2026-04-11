// Package auth provides JWT token generation and validation.
// It lives in internal/auth because it is domain-specific (tied to our User entity).
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType differentiates access tokens from refresh tokens in claims.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// Claims is our custom JWT payload.
type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"`
	TokenType TokenType `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTService holds the signing secrets and provides token operations.
type JWTService struct {
	accessSecret  string
	refreshSecret string
}

// NewJWTService constructs a JWTService.
func NewJWTService(accessSecret, refreshSecret string) *JWTService {
	return &JWTService{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
	}
}

// GenerateAccessToken creates a short-lived access token.
func (s *JWTService) GenerateAccessToken(userID uuid.UUID, role string, expiry time.Duration) (string, error) {
	return s.generate(userID, role, TokenTypeAccess, expiry, s.accessSecret)
}

// GenerateRefreshToken creates a long-lived refresh token.
func (s *JWTService) GenerateRefreshToken(userID uuid.UUID, expiry time.Duration) (string, error) {
	return s.generate(userID, "", TokenTypeRefresh, expiry, s.refreshSecret)
}

// generate is the internal signing helper.
func (s *JWTService) generate(
	userID uuid.UUID,
	role string,
	tokenType TokenType,
	expiry time.Duration,
	secret string,
) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		UserID:    userID,
		Role:      role,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("jwt.generate: %w", err)
	}
	return signed, nil
}

// ParseAccessToken validates and parses an access token.
func (s *JWTService) ParseAccessToken(tokenStr string) (*Claims, error) {
	return s.parse(tokenStr, s.accessSecret, TokenTypeAccess)
}

// ParseRefreshToken validates and parses a refresh token.
func (s *JWTService) ParseRefreshToken(tokenStr string) (*Claims, error) {
	return s.parse(tokenStr, s.refreshSecret, TokenTypeRefresh)
}

// parse validates the signature and type claim.
func (s *JWTService) parse(tokenStr, secret string, expectedType TokenType) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != expectedType {
		return nil, errors.New("wrong token type")
	}
	return claims, nil
}
