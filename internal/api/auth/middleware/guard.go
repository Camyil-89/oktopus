package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"oktopus/internal/api/platform/response"
	"oktopus/internal/db/user/domain"
)

// ActiveUsers проверка активной сессии по id пользователя.
type ActiveUsers interface {
	AssertActiveUser(ctx context.Context, userID uuid.UUID) error
}

// Guard JWT-сессия и проверка, что пользователь включён.
type Guard struct {
	Secret       []byte
	Users        ActiveUsers
	LoginLockout *LoginLockout
}

func NewGuard(secret []byte, users ActiveUsers, loginLockout *LoginLockout) *Guard {
	return &Guard{Secret: secret, Users: users, LoginLockout: loginLockout}
}

// Require оборачивает handler проверкой сессии и статуса учётной записи.
func (g *Guard) Require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if g.LoginLockout != nil && g.LoginLockout.Blocked(time.Now(), ClientIP(r)) {
			response.Error(w, http.StatusTooManyRequests, "authentication temporarily disabled")
			return
		}
		claims, ok := ClaimsFromRequest(r, g.Secret)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if err := g.Users.AssertActiveUser(r.Context(), userID); err != nil {
			if errors.Is(err, domain.ErrUserDisabled) {
				response.Error(w, http.StatusForbidden, "user disabled")
				return
			}
			if errors.Is(err, domain.ErrNotFound) {
				response.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			response.Error(w, http.StatusInternalServerError, "auth check failed")
			return
		}
		next(w, r)
	}
}
