package view

import (
	"net/http"

	"github.com/google/uuid"

	"oktopus/internal/api/platform/response"
	"oktopus/internal/apperr"
)

func parseInstanceID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	raw := r.PathValue("id")
	id, err := uuid.Parse(raw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, string(apperr.InstanceNotFound))
		return uuid.Nil, false
	}
	return id, true
}
