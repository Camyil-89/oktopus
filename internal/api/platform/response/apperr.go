package response

import (
	"net/http"

	"oktopus/internal/apperr"
)

// ErrorFrom пишет JSON error: код apperr или текст err.
func ErrorFrom(w http.ResponseWriter, status int, err error) {
	if err == nil {
		Error(w, status, "unknown error")
		return
	}
	Error(w, status, apperr.PublicMessage(err))
}

// StatusForErr выбирает HTTP-статус для apperr и типичных сценариев.
func StatusForErr(err error, clientStatus, serverStatus int) int {
	if apperr.IsClient(err) {
		return clientStatus
	}
	if _, ok := apperr.CodeOf(err); ok {
		return serverStatus
	}
	return clientStatus
}
