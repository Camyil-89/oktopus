package view

import "net/http"

// Register монтирует маршруты пользователей на mux.
func Register(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("GET /api/users", h.withAuth(h.List))
	mux.HandleFunc("POST /api/users", h.withAuth(h.Create))
	mux.HandleFunc("GET /api/users/{id}", h.withAuth(h.Get))
	mux.HandleFunc("PATCH /api/users/{id}/password", h.withAuth(h.ChangePassword))
	mux.HandleFunc("PATCH /api/users/{id}/enabled", h.withAuth(h.SetEnabled))
	mux.HandleFunc("DELETE /api/users/{id}", h.withAuth(h.Delete))
}
