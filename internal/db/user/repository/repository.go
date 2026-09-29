package repository

import (
	"context"

	"github.com/google/uuid"

	"oktopus/internal/db/user/domain"
)

// UserRepository персистентность пользователей (интерфейс доменного слоя).
type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	Count(ctx context.Context, search string) (int64, error)
	List(ctx context.Context, search string, limit, offset int32) ([]domain.User, error)
	Create(ctx context.Context, user domain.User) (domain.User, error)
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error
	UpdateEnabled(ctx context.Context, id uuid.UUID, enabled bool) error
	Delete(ctx context.Context, id uuid.UUID) error
}
