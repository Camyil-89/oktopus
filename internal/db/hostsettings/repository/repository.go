package repository

import (
	"context"

	"github.com/google/uuid"

	"oktopus/internal/db/hostsettings/domain"
)

type Repository interface {
	Get(ctx context.Context) (domain.Settings, error)
	Insert(ctx context.Context, s domain.Settings) (domain.Settings, error)
	Update(ctx context.Context, s domain.Settings) (domain.Settings, error)
	UpdateReportsDashboard(ctx context.Context, id uuid.UUID, body []byte) (domain.Settings, error)
}
