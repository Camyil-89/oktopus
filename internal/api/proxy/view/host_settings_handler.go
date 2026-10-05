package view

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	authmw "oktopus/internal/api/auth/middleware"
	"oktopus/internal/api/platform/response"
	"oktopus/internal/apperr"
	hostsettingsservice "oktopus/internal/db/hostsettings/service"
)

type HostSettingsHandler struct {
	host *hostsettingsservice.Service
	auth *authmw.Guard
}

func NewHostSettingsHandler(host *hostsettingsservice.Service, auth *authmw.Guard) *HostSettingsHandler {
	return &HostSettingsHandler{host: host, auth: auth}
}

func (h *HostSettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	st, err := h.host.Get(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, string(apperr.LoadProxySettingsFailed))
		return
	}
	response.JSON(w, http.StatusOK, hostSettingsResponse{
		ID:                     st.ID.String(),
		AccessLogRetentionDays: st.AccessLogRetentionDays,
		UpdatedAt:              st.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

func (h *HostSettingsHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var body hostSettingsPatch
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, string(apperr.InvalidJSON))
		return
	}
	updated, err := h.host.Update(r.Context(), hostsettingsservice.UpdateInput{
		AccessLogRetentionDays: body.AccessLogRetentionDays,
	})
	if err != nil {
		response.ErrorFrom(w, http.StatusBadRequest, err)
		return
	}
	response.JSON(w, http.StatusOK, hostSettingsResponse{
		ID:                     updated.ID.String(),
		AccessLogRetentionDays: updated.AccessLogRetentionDays,
		UpdatedAt:              updated.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

func (h *HostSettingsHandler) GetReportsDashboard(w http.ResponseWriter, r *http.Request) {
	body, err := h.host.ReportsDashboard(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, string(apperr.LoadReportsDashboardFailed))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (h *HostSettingsHandler) PatchReportsDashboard(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		response.Error(w, http.StatusBadRequest, string(apperr.InvalidJSON))
		return
	}
	updated, err := h.host.UpdateReportsDashboard(r.Context(), raw)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, string(apperr.UpdateReportsDashboardFailed))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(updated)
}

func (h *HostSettingsHandler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return h.auth.Require(next)
}

type hostSettingsResponse struct {
	ID                     string `json:"id"`
	AccessLogRetentionDays int32  `json:"access_log_retention_days"`
	UpdatedAt              string `json:"updated_at"`
}

type hostSettingsPatch struct {
	AccessLogRetentionDays *int32 `json:"access_log_retention_days"`
}
