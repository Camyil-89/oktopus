package view

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	authmw "oktopus/internal/api/auth/middleware"
	"oktopus/internal/api/platform/response"
	"oktopus/internal/db/proxysettings/domain"
	proxysettingsservice "oktopus/internal/db/proxysettings/service"
)

type Handler struct {
	settings *proxysettingsservice.Service
	auth     *authmw.Guard
}

func NewHandler(settings *proxysettingsservice.Service, auth *authmw.Guard) *Handler {
	return &Handler{settings: settings, auth: auth}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st, err := h.settings.Get(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "load proxy settings failed")
		return
	}
	response.JSON(w, http.StatusOK, toResponse(st))
}

func (h *Handler) Patch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body patchRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}

	updated, err := h.settings.Update(r.Context(), body.toInput())
	if err != nil {
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if strings.Contains(err.Error(), "apply proxy config") {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "update proxy settings failed")
		return
	}
	response.JSON(w, http.StatusOK, toResponse(updated))
}

func (h *Handler) CAStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st, err := h.settings.CAStatus(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "ca status failed")
		return
	}
	out := caStatusResponse{
		CertInstalled: st.CertInstalled,
		KeyInstalled:  st.KeyInstalled,
	}
	if st.ValidUntil != nil {
		out.ValidUntil = st.ValidUntil.UTC().Format(time.RFC3339)
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *Handler) DownloadCert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path, err := h.settings.CACertPath(r.Context())
	if err != nil {
		response.Error(w, http.StatusNotFound, "certificate not found")
		return
	}
	serveDownload(w, path, "ca.crt")
}

func (h *Handler) DownloadKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path, err := h.settings.CAKeyPath(r.Context())
	if err != nil {
		response.Error(w, http.StatusNotFound, "key not found")
		return
	}
	serveDownload(w, path, "ca.key")
}

func serveDownload(w http.ResponseWriter, path, filename string) {
	data, err := os.ReadFile(path)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "read file failed")
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	if strings.HasSuffix(strings.ToLower(path), ".key") {
		w.Header().Set("Content-Type", "application/x-pem-file")
	} else {
		w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) GenerateCA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body generateCARequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	updated, err := h.settings.GenerateCA(r.Context(), proxysettingsservice.GenerateCAInput{
		KeyBits:      body.KeyBits,
		CommonName:   body.CommonName,
		Organization: body.Organization,
		Country:      body.Country,
		ValidDays:    body.ValidDays,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, toResponse(updated))
}

func (h *Handler) UploadCA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	const maxBody = 512 << 10
	if err := r.ParseMultipartForm(maxBody); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	certFile, _, err := r.FormFile("cert")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "cert file required")
		return
	}
	defer certFile.Close()
	keyFile, _, err := r.FormFile("key")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "key file required")
		return
	}
	defer keyFile.Close()

	updated, err := h.settings.UploadCA(r.Context(), certFile, keyFile)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, toResponse(updated))
}

func (h *Handler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return h.auth.Require(next)
}

func toResponse(st domain.Settings) settingsResponse {
	return settingsResponse{
		ID:                    st.ID.String(),
		ProxyEnabled:          st.ProxyEnabled,
		Listen:                st.Listen,
		ConnectMode:           st.ConnectMode,
		AuthEnabled:           st.AuthEnabled,
		AuthStaticUsers:       st.AuthStaticUsers,
		AuthRealm:             st.AuthRealm,
		AuthBackend:           st.AuthBackend,
		AuthCacheTTLMinutes:   st.AuthCacheTTLMinutes,
		LDAPURL:             st.LDAPURL,
		LDAPBaseDN:          st.LDAPBaseDN,
		LDAPBindDN:          st.LDAPBindDN,
		LDAPBindPasswordSet:    st.LDAPBindPassword != "",
		AccessLogRetentionDays: st.AccessLogRetentionDays,
		UpdatedAt:              st.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type settingsResponse struct {
	ID                    string `json:"id"`
	ProxyEnabled          bool   `json:"proxy_enabled"`
	Listen                string `json:"listen"`
	ConnectMode           string `json:"connect_mode"`
	AuthEnabled           bool   `json:"auth_enabled"`
	AuthStaticUsers       string `json:"auth_static_users"`
	AuthRealm             string `json:"auth_realm"`
	AuthBackend           string `json:"auth_backend"`
	AuthCacheTTLMinutes   int64  `json:"auth_cache_ttl_minutes"`
	LDAPURL               string `json:"ldap_url"`
	LDAPBaseDN            string `json:"ldap_base_dn"`
	LDAPBindDN            string `json:"ldap_bind_dn"`
	LDAPBindPasswordSet      bool  `json:"ldap_bind_password_set"`
	AccessLogRetentionDays int32 `json:"access_log_retention_days"`
	UpdatedAt                string `json:"updated_at"`
}

type caStatusResponse struct {
	CertInstalled bool   `json:"cert_installed"`
	KeyInstalled  bool   `json:"key_installed"`
	ValidUntil    string `json:"valid_until,omitempty"`
}

type patchRequest struct {
	ProxyEnabled         *bool   `json:"proxy_enabled"`
	Listen               *string `json:"listen"`
	ConnectMode          *string `json:"connect_mode"`
	AuthEnabled          *bool   `json:"auth_enabled"`
	AuthStaticUsers      *string `json:"auth_static_users"`
	AuthRealm            *string `json:"auth_realm"`
	AuthBackend          *string `json:"auth_backend"`
	AuthCacheTTLMinutes  *int64  `json:"auth_cache_ttl_minutes"`
	LDAPURL              *string `json:"ldap_url"`
	LDAPBaseDN           *string `json:"ldap_base_dn"`
	LDAPBindDN           *string `json:"ldap_bind_dn"`
	LDAPBindPassword         *string `json:"ldap_bind_password"`
	AccessLogRetentionDays   *int32  `json:"access_log_retention_days"`
}

func (p patchRequest) toInput() proxysettingsservice.UpdateInput {
	return proxysettingsservice.UpdateInput{
		ProxyEnabled:        p.ProxyEnabled,
		Listen:              p.Listen,
		ConnectMode:         p.ConnectMode,
		AuthEnabled:         p.AuthEnabled,
		AuthStaticUsers:     p.AuthStaticUsers,
		AuthRealm:           p.AuthRealm,
		AuthBackend:         p.AuthBackend,
		AuthCacheTTLMinutes: p.AuthCacheTTLMinutes,
		LDAPURL:             p.LDAPURL,
		LDAPBaseDN:          p.LDAPBaseDN,
		LDAPBindDN:          p.LDAPBindDN,
		LDAPBindPassword:         p.LDAPBindPassword,
		AccessLogRetentionDays:   p.AccessLogRetentionDays,
	}
}

type generateCARequest struct {
	KeyBits      int    `json:"key_bits"`
	CommonName   string `json:"common_name"`
	Organization string `json:"organization"`
	Country      string `json:"country"`
	ValidDays    int    `json:"valid_days"`
}
