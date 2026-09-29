package response

import (
	"encoding/json"
	"net/http"
)

type errorBody struct {
	Error        string `json:"error"`
	Diagnostics  any    `json:"diagnostics,omitempty"`
}

// JSON пишет JSON-ответ с заданным статусом.
func JSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error пишет JSON с полем error.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, errorBody{Error: msg})
}

// ErrorWithDiagnostics пишет JSON с error и списком диагностик.
func ErrorWithDiagnostics(w http.ResponseWriter, status int, msg string, diagnostics any) {
	JSON(w, status, errorBody{Error: msg, Diagnostics: diagnostics})
}
