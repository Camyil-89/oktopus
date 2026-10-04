package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"oktopus/internal/db/hostsettings/domain"
	"oktopus/internal/db/hostsettings/repository"
	"oktopus/internal/db/infrastructure/postgres/store"
)

type SettingsRepository struct {
	q *store.Queries
}

func New(q *store.Queries) repository.Repository {
	return &SettingsRepository{q: q}
}

func (r *SettingsRepository) Get(ctx context.Context) (domain.Settings, error) {
	row, err := r.q.GetSettings(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Settings{}, domain.ErrNotFound
		}
		return domain.Settings{}, fmt.Errorf("postgres get settings: %w", err)
	}
	return toDomain(row), nil
}

func (r *SettingsRepository) Insert(ctx context.Context, s domain.Settings) (domain.Settings, error) {
	row, err := r.q.InsertSettings(ctx, store.InsertSettingsParams{
		ID:                     s.ID,
		AccessLogRetentionDays: s.AccessLogRetentionDays,
	})
	if err != nil {
		return domain.Settings{}, fmt.Errorf("postgres insert settings: %w", err)
	}
	return toDomain(row), nil
}

func (r *SettingsRepository) Update(ctx context.Context, s domain.Settings) (domain.Settings, error) {
	row, err := r.q.UpdateSettings(ctx, store.UpdateSettingsParams{
		ID:                     s.ID,
		AccessLogRetentionDays: s.AccessLogRetentionDays,
	})
	if err != nil {
		return domain.Settings{}, fmt.Errorf("postgres update settings: %w", err)
	}
	return toDomain(row), nil
}

func toDomain(row store.Setting) domain.Settings {
	return domain.Settings{
		ID:                     row.ID,
		AccessLogRetentionDays: row.AccessLogRetentionDays,
		CreatedAt:              row.CreatedAt.Time,
		UpdatedAt:              row.UpdatedAt.Time,
	}
}
