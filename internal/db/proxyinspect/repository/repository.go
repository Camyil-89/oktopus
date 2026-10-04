package repository

import (
	"context"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyinspect/domain"
)

type RulesRepository interface {
	ListByInstance(ctx context.Context, instanceID uuid.UUID) ([]domain.Rule, error)
	ListSummaryByInstance(ctx context.Context, instanceID uuid.UUID) ([]domain.RuleSummary, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Rule, error)
	CountByInstance(ctx context.Context, instanceID uuid.UUID) (int64, error)
	ReplaceAllForInstance(ctx context.Context, instanceID uuid.UUID, rules []domain.Rule) error
	ListInstanceIDs(ctx context.Context) ([]uuid.UUID, error)
}
