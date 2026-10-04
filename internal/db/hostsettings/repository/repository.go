package repository

import (
	"context"

	"oktopus/internal/db/hostsettings/domain"
)

type Repository interface {
	Get(ctx context.Context) (domain.Settings, error)
	Insert(ctx context.Context, s domain.Settings) (domain.Settings, error)
	Update(ctx context.Context, s domain.Settings) (domain.Settings, error)
}
