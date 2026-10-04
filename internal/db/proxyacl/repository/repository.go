package repository

import (
	"context"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyacl/domain"
)

type RulesRepository interface {
	GetPolicy(ctx context.Context, instanceID uuid.UUID) (domain.Policy, error)
	SetPolicy(ctx context.Context, instanceID uuid.UUID, configText string) (domain.Policy, error)
	ListPolicyInstanceIDs(ctx context.Context) ([]uuid.UUID, error)
	ListNamedLists(ctx context.Context) ([]domain.NamedList, error)
	ListNamedListsSummary(ctx context.Context) ([]domain.NamedListSummary, error)
	GetNamedList(ctx context.Context, id uuid.UUID) (domain.NamedList, error)
	ListNamedListsByNames(ctx context.Context, names []string) ([]domain.NamedList, error)
	ReplaceNamedLists(ctx context.Context, lists []domain.NamedList) error
	UpdateNamedListBody(ctx context.Context, id uuid.UUID, body string) (domain.NamedList, error)
}
