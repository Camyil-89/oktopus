package auth

import (
	"context"
	"errors"
)

// Identity — результат успешной проверки (для логов и ACL).
type Identity struct {
	Username string
	Groups   []string // CN групп LDAP; static — пусто
}

// Authenticator проверяет учётные данные proxy-клиента.
// Реализации: Static (файл), позже LDAP и др.
type Authenticator interface {
	Authenticate(ctx context.Context, username, password string) (Identity, error)
}

// ErrInvalidCredentials — неверный логин или пароль.
type ErrInvalidCredentials struct{}

func (ErrInvalidCredentials) Error() string { return "invalid proxy credentials" }

// IsInvalidCredentials сообщает, что ошибка — отказ в доступе, а не сбой backend.
func IsInvalidCredentials(err error) bool {
	var ic ErrInvalidCredentials
	return errors.As(err, &ic)
}
