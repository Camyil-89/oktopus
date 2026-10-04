package repository

import (
	"context"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyinstances/domain"
)

type Repository interface {
	List(ctx context.Context) ([]domain.Instance, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Instance, error)
	Insert(ctx context.Context, inst domain.Instance) (domain.Instance, error)
	Update(ctx context.Context, inst domain.Instance) (domain.Instance, error)
	UpdateCAPaths(ctx context.Context, id uuid.UUID, certPath, keyPath string) (domain.Instance, error)
	Delete(ctx context.Context, id uuid.UUID) error
	InsertACLPolicy(ctx context.Context, policyID, instanceID uuid.UUID) error
}
