package repository

import (
	"context"

	"github.com/google/uuid"

	"oktopus/internal/db/proxysettings/domain"
)

// SettingsRepository персистентность настроек прокси.
type SettingsRepository interface {
	Get(ctx context.Context) (domain.Settings, error)
	Insert(ctx context.Context, s domain.Settings) (domain.Settings, error)
	Update(ctx context.Context, s domain.Settings) (domain.Settings, error)
	UpdateCAPaths(ctx context.Context, id uuid.UUID, certPath, keyPath string) (domain.Settings, error)
}
