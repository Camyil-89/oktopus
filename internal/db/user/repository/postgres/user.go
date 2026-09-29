package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"oktopus/internal/db/infrastructure/postgres/store"
	"oktopus/internal/db/user/domain"
	"oktopus/internal/db/user/repository"
)

// UserRepository реализует repository.UserRepository через sqlc.
type UserRepository struct {
	q *store.Queries
}

func NewUserRepository(q *store.Queries) repository.UserRepository {
	return &UserRepository{q: q}
}

func (r *UserRepository) FindByUsername(ctx context.Context, username string) (domain.User, error) {
	row, err := r.q.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("postgres get user by username: %w", err)
	}
	return toDomain(row), nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("postgres get user by id: %w", err)
	}
	return toDomain(row), nil
}

func (r *UserRepository) Count(ctx context.Context, search string) (int64, error) {
	n, err := r.q.CountUsers(ctx, search)
	if err != nil {
		return 0, fmt.Errorf("postgres count users: %w", err)
	}
	return n, nil
}

func (r *UserRepository) List(ctx context.Context, search string, limit, offset int32) ([]domain.User, error) {
	rows, err := r.q.ListUsers(ctx, store.ListUsersParams{
		Search:    search,
		LimitVal:  limit,
		OffsetVal: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("postgres list users: %w", err)
	}
	out := make([]domain.User, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDomain(row))
	}
	return out, nil
}

func (r *UserRepository) Create(ctx context.Context, user domain.User) (domain.User, error) {
	row, err := r.q.CreateUser(ctx, store.CreateUserParams{
		ID:           user.ID,
		Username:     user.Username,
		PasswordHash: user.PasswordHash,
		Enabled:      user.Enabled,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, domain.ErrUsernameTaken
		}
		return domain.User{}, fmt.Errorf("postgres create user: %w", err)
	}
	return toDomain(row), nil
}

func (r *UserRepository) UpdateEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	err := r.q.UpdateUserEnabled(ctx, store.UpdateUserEnabledParams{
		ID:      id,
		Enabled: enabled,
	})
	if err != nil {
		return fmt.Errorf("postgres update user enabled: %w", err)
	}
	return nil
}

func (r *UserRepository) UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	err := r.q.UpdateUserPassword(ctx, store.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return fmt.Errorf("postgres update password: %w", err)
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.q.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("postgres delete user: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

func toDomain(u store.User) domain.User {
	return domain.User{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Enabled:      u.Enabled,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}
