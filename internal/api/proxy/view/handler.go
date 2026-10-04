package view

import (
	"net/http"

	authmw "oktopus/internal/api/auth/middleware"
)

// Handler — глобальные страницы forbidden/gateway (методы в forbidden_handler.go, gateway_handler.go).
type Handler struct {
	auth *authmw.Guard
}

func NewHandler(auth *authmw.Guard) *Handler {
	return &Handler{auth: auth}
}

func (h *Handler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return h.auth.Require(next)
}
