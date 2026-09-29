package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound           = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrProtectedUser      = errors.New("protected user cannot be deleted")
	ErrUserDisabled       = errors.New("user disabled")
	ErrUsernameTaken      = errors.New("username already taken")
)

// User учётная запись в доменной модели.
type User struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
