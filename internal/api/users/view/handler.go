package view

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	authmw "oktopus/internal/api/auth/middleware"
	"oktopus/internal/api/platform/response"
	"oktopus/internal/db/user/domain"
	userservice "oktopus/internal/db/user/service"
)

// Handler HTTP-слой управления пользователями.
type Handler struct {
	users *userservice.Service
	auth  *authmw.Guard
}

func NewHandler(users *userservice.Service, auth *authmw.Guard) *Handler {
	return &Handler{
		users: users,
		auth:  auth,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	pageData, err := h.users.List(r.Context(), search, page, pageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "list users failed")
		return
	}

	results := make([]userResponse, 0, len(pageData.Items))
	for _, u := range pageData.Items {
		results = append(results, toUserResponse(u, h.users.IsProtectedUsername(u.Username)))
	}

	response.JSON(w, http.StatusOK, listResponse{
		Count:    pageData.Total,
		Page:     pageData.Page,
		PageSize: pageData.Size,
		Results:  results,
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	user, err := h.users.CreateWithPassword(r.Context(), body.Username, body.Password)
	if err != nil {
		if errors.Is(err, domain.ErrUsernameTaken) {
			response.Error(w, http.StatusConflict, "username already taken")
			return
		}
		if strings.Contains(err.Error(), "required") {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "create user failed")
		return
	}

	response.JSON(w, http.StatusCreated, toUserResponse(user, h.users.IsProtectedUsername(user.Username)))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, ok := parseUserID(r.PathValue("id"))
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := h.users.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "user not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "get user failed")
		return
	}

	response.JSON(w, http.StatusOK, toUserResponse(user, h.users.IsProtectedUsername(user.Username)))
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, ok := parseUserID(r.PathValue("id"))
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var body changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := h.users.UpdatePassword(r.Context(), id, body.Password); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "user not found")
			return
		}
		if strings.Contains(err.Error(), "required") {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "change password failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetEnabled(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, ok := parseUserID(r.PathValue("id"))
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var body setEnabledRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := h.users.SetEnabled(r.Context(), id, body.Enabled); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "user not found")
			return
		}
		if errors.Is(err, domain.ErrProtectedUser) {
			response.Error(w, http.StatusForbidden, "protected user cannot be disabled")
			return
		}
		response.Error(w, http.StatusInternalServerError, "set enabled failed")
		return
	}

	user, err := h.users.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "get user failed")
		return
	}

	response.JSON(w, http.StatusOK, toUserResponse(user, h.users.IsProtectedUsername(user.Username)))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	id, ok := parseUserID(r.PathValue("id"))
	if !ok {
		response.Error(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.users.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "user not found")
			return
		}
		if errors.Is(err, domain.ErrProtectedUser) {
			response.Error(w, http.StatusForbidden, "protected user cannot be deleted")
			return
		}
		response.Error(w, http.StatusInternalServerError, "delete user failed")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return h.auth.Require(next)
}

func parseUserID(raw string) (uuid.UUID, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.UUID{}, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, false
	}
	return id, true
}

func toUserResponse(u domain.User, protected bool) userResponse {
	return userResponse{
		ID:        u.ID.String(),
		Username:  u.Username,
		Protected: protected,
		Enabled:   u.Enabled,
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
