package view

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	authmw "oktopus/internal/api/auth/middleware"
	"oktopus/internal/api/platform/response"
	"oktopus/internal/apperr"
	"oktopus/internal/db/proxyinstances/domain"
	proxyinstancesservice "oktopus/internal/db/proxyinstances/service"
)

type InstancesHandler struct {
	instances *proxyinstancesservice.Service
	auth      *authmw.Guard
}

func NewInstancesHandler(instances *proxyinstancesservice.Service, auth *authmw.Guard) *InstancesHandler {
	return &InstancesHandler{instances: instances, auth: auth}
}

func (h *InstancesHandler) List(w http.ResponseWriter, r *http.Request) {
	list, err := h.instances.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, string(apperr.LoadProxySettingsFailed))
		return
	}
	out := make([]instanceResponse, len(list))
	for i := range list {
		out[i] = toInstanceResponse(list[i])
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *InstancesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, string(apperr.InvalidJSON))
		return
	}
	inst, err := h.instances.Create(r.Context(), proxyinstancesservice.CreateInput{
		Name:   body.Name,
		Listen: body.Listen,
	})
	if err != nil {
		response.ErrorFrom(w, http.StatusBadRequest, err)
		return
	}
	response.JSON(w, http.StatusCreated, toInstanceResponse(inst))
}

func (h *InstancesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInstanceID(w, r)
	if !ok {
		return
	}
	inst, err := h.instances.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, string(apperr.InstanceNotFound))
			return
		}
		response.Error(w, http.StatusInternalServerError, string(apperr.LoadProxySettingsFailed))
		return
	}
	response.JSON(w, http.StatusOK, toInstanceResponse(inst))
}

func (h *InstancesHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInstanceID(w, r)
	if !ok {
		return
	}
	var body patchInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, string(apperr.InvalidJSON))
		return
	}
	updated, err := h.instances.Update(r.Context(), id, body.toInput())
	if err != nil {
		response.ErrorFrom(w, http.StatusBadRequest, err)
		return
	}
	response.JSON(w, http.StatusOK, toInstanceResponse(updated))
}

func (h *InstancesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInstanceID(w, r)
	if !ok {
		return
	}
	if err := h.instances.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, string(apperr.InstanceNotFound))
			return
		}
		response.Error(w, http.StatusInternalServerError, string(apperr.UpdateProxySettingsFailed))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *InstancesHandler) CAStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInstanceID(w, r)
	if !ok {
		return
	}
	st, err := h.instances.CAStatus(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, string(apperr.CAStatusFailed))
		return
	}
	out := caStatusResponse{CertInstalled: st.CertInstalled, KeyInstalled: st.KeyInstalled}
	if st.ValidUntil != nil {
		out.ValidUntil = st.ValidUntil.UTC().Format(time.RFC3339)
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *InstancesHandler) DownloadCert(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInstanceID(w, r)
	if !ok {
		return
	}
	path, err := h.instances.CACertPath(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, string(apperr.CertNotFound))
		return
	}
	serveDownload(w, path, "ca.crt")
}

func (h *InstancesHandler) DownloadKey(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInstanceID(w, r)
	if !ok {
		return
	}
	path, err := h.instances.CAKeyPath(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, string(apperr.KeyNotFound))
		return
	}
	serveDownload(w, path, "ca.key")
}

func (h *InstancesHandler) GenerateCA(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInstanceID(w, r)
	if !ok {
		return
	}
	var body generateCARequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, string(apperr.InvalidJSON))
		return
	}
	updated, err := h.instances.GenerateCA(r.Context(), id, proxyinstancesservice.GenerateCAInput{
		KeyBits: body.KeyBits, CommonName: body.CommonName, Organization: body.Organization,
		Country: body.Country, ValidDays: body.ValidDays,
	})
	if err != nil {
		response.ErrorFrom(w, http.StatusBadRequest, err)
		return
	}
	response.JSON(w, http.StatusOK, toInstanceResponse(updated))
}

func (h *InstancesHandler) UploadCA(w http.ResponseWriter, r *http.Request) {
	id, ok := parseInstanceID(w, r)
	if !ok {
		return
	}
	const maxBody = 512 << 10
	if err := r.ParseMultipartForm(maxBody); err != nil {
		response.Error(w, http.StatusBadRequest, string(apperr.InvalidMultipartForm))
		return
	}
	certFile, _, err := r.FormFile("cert")
	if err != nil {
		response.Error(w, http.StatusBadRequest, string(apperr.CertFileRequired))
		return
	}
	defer certFile.Close()
	keyFile, _, err := r.FormFile("key")
	if err != nil {
		response.Error(w, http.StatusBadRequest, string(apperr.KeyFileRequired))
		return
	}
	defer keyFile.Close()
	updated, err := h.instances.UploadCA(r.Context(), id, certFile, keyFile)
	if err != nil {
		response.ErrorFrom(w, http.StatusBadRequest, err)
		return
	}
	response.JSON(w, http.StatusOK, toInstanceResponse(updated))
}

func (h *InstancesHandler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return h.auth.Require(next)
}

func serveDownload(w http.ResponseWriter, path, filename string) {
	data, err := os.ReadFile(path)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, string(apperr.ReadFileFailed))
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

type instanceResponse struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Enabled             bool   `json:"enabled"`
	Listen              string `json:"listen"`
	ConnectMode         string `json:"connect_mode"`
	AuthEnabled         bool   `json:"auth_enabled"`
	AuthStaticUsers     string `json:"auth_static_users"`
	AuthRealm           string `json:"auth_realm"`
	AuthBackend         string `json:"auth_backend"`
	AuthCacheTTLMinutes int64  `json:"auth_cache_ttl_minutes"`
	LDAPURL             string `json:"ldap_url"`
	LDAPBaseDN          string `json:"ldap_base_dn"`
	LDAPBindDN          string `json:"ldap_bind_dn"`
	LDAPBindPasswordSet bool   `json:"ldap_bind_password_set"`
	SortOrder           int    `json:"sort_order"`
	UpdatedAt           string `json:"updated_at"`
}

func toInstanceResponse(inst domain.Instance) instanceResponse {
	return instanceResponse{
		ID: inst.ID.String(), Name: inst.Name, Enabled: inst.Enabled, Listen: inst.Listen,
		ConnectMode: inst.ConnectMode, AuthEnabled: inst.AuthEnabled, AuthStaticUsers: inst.AuthStaticUsers,
		AuthRealm: inst.AuthRealm, AuthBackend: inst.AuthBackend, AuthCacheTTLMinutes: inst.AuthCacheTTLMinutes,
		LDAPURL: inst.LDAPURL, LDAPBaseDN: inst.LDAPBaseDN, LDAPBindDN: inst.LDAPBindDN,
		LDAPBindPasswordSet: inst.LDAPBindPassword != "", SortOrder: inst.SortOrder,
		UpdatedAt: inst.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type createInstanceRequest struct {
	Name   string `json:"name"`
	Listen string `json:"listen"`
}

type patchInstanceRequest struct {
	Name                *string `json:"name"`
	Enabled             *bool   `json:"enabled"`
	Listen              *string `json:"listen"`
	ConnectMode         *string `json:"connect_mode"`
	AuthEnabled         *bool   `json:"auth_enabled"`
	AuthStaticUsers     *string `json:"auth_static_users"`
	AuthRealm           *string `json:"auth_realm"`
	AuthBackend         *string `json:"auth_backend"`
	AuthCacheTTLMinutes *int64  `json:"auth_cache_ttl_minutes"`
	LDAPURL             *string `json:"ldap_url"`
	LDAPBaseDN          *string `json:"ldap_base_dn"`
	LDAPBindDN          *string `json:"ldap_bind_dn"`
	LDAPBindPassword    *string `json:"ldap_bind_password"`
	SortOrder           *int    `json:"sort_order"`
}

func (p patchInstanceRequest) toInput() proxyinstancesservice.UpdateInput {
	return proxyinstancesservice.UpdateInput{
		Name: p.Name, Enabled: p.Enabled, Listen: p.Listen, ConnectMode: p.ConnectMode,
		AuthEnabled: p.AuthEnabled, AuthStaticUsers: p.AuthStaticUsers, AuthRealm: p.AuthRealm,
		AuthBackend: p.AuthBackend, AuthCacheTTLMinutes: p.AuthCacheTTLMinutes,
		LDAPURL: p.LDAPURL, LDAPBaseDN: p.LDAPBaseDN, LDAPBindDN: p.LDAPBindDN,
		LDAPBindPassword: p.LDAPBindPassword, SortOrder: p.SortOrder,
	}
}

type generateCARequest struct {
	KeyBits      int    `json:"key_bits"`
	CommonName   string `json:"common_name"`
	Organization string `json:"organization"`
	Country      string `json:"country"`
	ValidDays    int    `json:"valid_days"`
}

type caStatusResponse struct {
	CertInstalled bool   `json:"cert_installed"`
	KeyInstalled  bool   `json:"key_installed"`
	ValidUntil    string `json:"valid_until,omitempty"`
}
