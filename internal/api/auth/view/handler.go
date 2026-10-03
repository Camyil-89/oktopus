package view

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	authmw "oktopus/internal/api/auth/middleware"
	"oktopus/internal/apperr"
	"oktopus/internal/api/platform/response"
	"oktopus/internal/auth"
	"oktopus/internal/db/user/domain"
	userservice "oktopus/internal/db/user/service"
)

// Handler HTTP-слой домена auth.
type Handler struct {
	users        *userservice.Service
	auth         *authmw.Guard
	loginLockout *authmw.LoginLockout
	cookieSecure bool
}

func NewHandler(users *userservice.Service, auth *authmw.Guard, loginLockout *authmw.LoginLockout, cookieSecure bool) *Handler {
	return &Handler{
		users:        users,
		auth:         auth,
		loginLockout: loginLockout,
		cookieSecure: cookieSecure,
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body loginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	if body.Username == "" || body.Password == "" {
		response.Error(w, http.StatusBadRequest, string(apperr.UsernamePasswordRequired))
		return
	}

	clientIP := authmw.ClientIP(r)
	now := time.Now()
	if h.loginLockout.Blocked(now, clientIP) {
		response.Error(w, http.StatusTooManyRequests, string(apperr.LoginTemporarilyDisabled))
		return
	}

	user, err := h.users.Authenticate(r.Context(), body.Username, body.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			h.loginLockout.RecordFailure(time.Now(), clientIP)
			response.Error(w, http.StatusUnauthorized, string(apperr.InvalidCredentials))
			return
		}
		if errors.Is(err, domain.ErrUserDisabled) {
			response.Error(w, http.StatusForbidden, string(apperr.UserDisabled))
			return
		}
		response.Error(w, http.StatusInternalServerError, string(apperr.LoginFailed))
		return
	}

	token, expiresAt, err := auth.IssueToken(h.auth.Secret, user.ID, user.Username)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, string(apperr.LoginFailed))
		return
	}
	auth.SetSessionCookie(w, token, expiresAt, h.cookieSecure)
	response.JSON(w, http.StatusOK, userResponse{ID: user.ID.String(), Username: user.Username})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	auth.ClearSessionCookie(w, h.cookieSecure)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	h.auth.Require(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := authmw.ClaimsFromRequest(r, h.auth.Secret)
		if !ok {
			response.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		response.JSON(w, http.StatusOK, userResponse{ID: claims.UserID, Username: claims.Username})
	})(w, r)
}
