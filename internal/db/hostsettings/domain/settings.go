package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("host settings not found")

type Settings struct {
	ID                     uuid.UUID
	AccessLogRetentionDays int32
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func Default(id uuid.UUID) Settings {
	return Settings{
		ID:                     id,
		AccessLogRetentionDays: 3,
	}
}
