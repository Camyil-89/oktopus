package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"oktopus/internal/db/user/domain"
	"oktopus/internal/db/user/repository"
	"oktopus/internal/id"
)

const defaultListPageSize = 20
const maxListPageSize = 100

// Service доменная логика пользователей.
type Service struct {
	repo                   repository.UserRepository
	protectedAdminUsername string
}

func New(repo repository.UserRepository, protectedAdminUsername string) *Service {
	return &Service{
		repo:                   repo,
		protectedAdminUsername: strings.TrimSpace(protectedAdminUsername),
	}
}

// ListPage страница пользователей (page начиная с 1).
type ListPage struct {
	Items []domain.User
	Total int64
	Page  int
	Size  int
}

// EnsureAdmin создаёт admin или обновляет его пароль до значения из конфигурации (каждый запуск).
func (s *Service) EnsureAdmin(ctx context.Context, username, plainPassword string) error {
	if username == "" {
		return fmt.Errorf("admin username is empty")
	}
	if plainPassword == "" {
		return fmt.Errorf("admin password is empty")
	}

	user, err := s.repo.FindByUsername(ctx, username)
	switch {
	case errors.Is(err, domain.ErrNotFound):
		_, err = s.CreateWithPassword(ctx, username, plainPassword)
		return err
	case err != nil:
		return err
	default:
		return s.UpdatePassword(ctx, user.ID, plainPassword)
	}
}

// List возвращает страницу пользователей.
func (s *Service) List(ctx context.Context, search string, page, pageSize int) (ListPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultListPageSize
	}
	if pageSize > maxListPageSize {
		pageSize = maxListPageSize
	}

	search = strings.TrimSpace(search)
	total, err := s.repo.Count(ctx, search)
	if err != nil {
		return ListPage{}, err
	}

	offset := int32((page - 1) * pageSize)
	items, err := s.repo.List(ctx, search, int32(pageSize), offset)
	if err != nil {
		return ListPage{}, err
	}

	return ListPage{
		Items: items,
		Total: total,
		Page:  page,
		Size:  pageSize,
	}, nil
}

// GetByID возвращает пользователя или domain.ErrNotFound.
func (s *Service) GetByID(ctx context.Context, userID uuid.UUID) (domain.User, error) {
	return s.repo.FindByID(ctx, userID)
}

// CreateWithPassword регистрирует пользователя с хешированным паролем.
func (s *Service) CreateWithPassword(ctx context.Context, username, plainPassword string) (domain.User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return domain.User{}, fmt.Errorf("username is required")
	}
	if plainPassword == "" {
		return domain.User{}, fmt.Errorf("password is required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}

	userID, err := id.New()
	if err != nil {
		return domain.User{}, fmt.Errorf("generate user id: %w", err)
	}

	return s.repo.Create(ctx, domain.User{
		ID:           userID,
		Username:     username,
		PasswordHash: string(hash),
		Enabled:      true,
	})
}

// UpdatePassword устанавливает новый пароль.
func (s *Service) UpdatePassword(ctx context.Context, userID uuid.UUID, plainPassword string) error {
	if plainPassword == "" {
		return fmt.Errorf("password is required")
	}
	_, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.repo.UpdatePasswordHash(ctx, userID, string(hash))
}

// SetEnabled включает или отключает пользователя; защищённый admin не отключается.
func (s *Service) SetEnabled(ctx context.Context, userID uuid.UUID, enabled bool) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if !enabled && s.isProtected(user.Username) {
		return domain.ErrProtectedUser
	}
	return s.repo.UpdateEnabled(ctx, userID, enabled)
}

// AssertActiveUser проверяет, что учётная запись существует и включена.
func (s *Service) AssertActiveUser(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if !user.Enabled {
		return domain.ErrUserDisabled
	}
	return nil
}

// Delete удаляет пользователя; защищённый admin не удаляется.
func (s *Service) Delete(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if s.isProtected(user.Username) {
		return domain.ErrProtectedUser
	}
	return s.repo.Delete(ctx, userID)
}

// IsProtectedUsername сообщает, защищён ли логин от удаления.
func (s *Service) IsProtectedUsername(username string) bool {
	return s.isProtected(username)
}

func (s *Service) isProtected(username string) bool {
	if s.protectedAdminUsername == "" {
		return false
	}
	return username == s.protectedAdminUsername
}

// GetByUsername возвращает пользователя или domain.ErrNotFound.
func (s *Service) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	return s.repo.FindByUsername(ctx, username)
}

// Authenticate проверяет логин и пароль.
func (s *Service) Authenticate(ctx context.Context, username, plainPassword string) (domain.User, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.User{}, domain.ErrInvalidCredentials
		}
		return domain.User{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(plainPassword)); err != nil {
		return domain.User{}, domain.ErrInvalidCredentials
	}
	if !user.Enabled {
		return domain.User{}, domain.ErrUserDisabled
	}
	return user, nil
}
