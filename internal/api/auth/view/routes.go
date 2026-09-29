package view

import "net/http"

// Register монтирует маршруты auth на mux.
func Register(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("/api/auth/login", h.Login)
	mux.HandleFunc("/api/auth/logout", h.Logout)
	mux.HandleFunc("/api/auth/me", h.Me)
}
