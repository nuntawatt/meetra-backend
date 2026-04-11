package entity

import (
	"time"

	"github.com/google/uuid"
)

// Role defines the permission level for a user.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// User is the core domain model.
// It contains zero framework dependencies and is used across all layers.
type User struct {
	ID        uuid.UUID  `db:"id"         json:"id"`
	Username  string     `db:"username"   json:"username"`
	Email     string     `db:"email"      json:"email"`
	Password  string     `db:"password"   json:"-"` // never serialised
	Role      Role       `db:"role"        json:"role"`
	AvatarURL string     `db:"avatar_url" json:"avatar_url,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// IsAdmin returns true if the user belongs to the admin role.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
