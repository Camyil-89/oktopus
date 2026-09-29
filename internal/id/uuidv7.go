package id

import (
	"fmt"

	"github.com/google/uuid"
)

// New генерирует UUID версии 7 (временная метка в начале, RFC 9562).
func New() (uuid.UUID, error) {
	return uuid.NewV7()
}

// MustNew как New, но паникует при ошибке (для инициализации, где сбой генерации недопустим).
func MustNew() uuid.UUID {
	u, err := New()
	if err != nil {
		panic(fmt.Sprintf("id: uuid v7: %v", err))
	}
	return u
}
