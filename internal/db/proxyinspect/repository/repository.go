package repository

import (
	"context"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyinspect/domain"
)

type RulesRepository interface {
	List(ctx context.Context) ([]domain.Rule, error)
	ListSummary(ctx context.Context) ([]domain.RuleSummary, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.Rule, error)
	Count(ctx context.Context) (int64, error)
	ReplaceAll(ctx context.Context, rules []domain.Rule) error
}
