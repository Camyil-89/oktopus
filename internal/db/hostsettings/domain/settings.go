package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("host settings not found")

// DefaultReportsDashboard — пустой документ вкладок отчётов (version 1).
var DefaultReportsDashboard = []byte(`{"version":1,"tabs":[]}`)

type Settings struct {
	ID                     uuid.UUID
	AccessLogRetentionDays int32
	ReportsDashboard       []byte
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func Default(id uuid.UUID) Settings {
	return Settings{
		ID:                     id,
		AccessLogRetentionDays: 3,
		ReportsDashboard:       DefaultReportsDashboard,
	}
}
