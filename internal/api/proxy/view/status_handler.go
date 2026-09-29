package view

import (
	"net/http"

	authmw "oktopus/internal/api/auth/middleware"
	"oktopus/internal/api/platform/response"
	proxyaclservice "oktopus/internal/db/proxyacl/service"
	proxysettingsservice "oktopus/internal/db/proxysettings/service"
	"oktopus/internal/proxy/metrics"
	"oktopus/internal/startup"
)

type StatusProvider interface {
	ProxyActive() bool
	ProxyListen() string
	ProxyLastStartError() string
	ProxyTraffic() metrics.Snapshot
}

type StatusHandler struct {
	settings *proxysettingsservice.Service
	acl      *proxyaclservice.Service
	proxy    StatusProvider
	auth     *authmw.Guard
}

func NewStatusHandler(
	settings *proxysettingsservice.Service,
	acl *proxyaclservice.Service,
	proxy StatusProvider,
	auth *authmw.Guard,
) *StatusHandler {
	return &StatusHandler{
		settings: settings,
		acl:      acl,
		proxy:    proxy,
		auth:     auth,
	}
}

func (h *StatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	aclSt, err := h.acl.CompileStatus(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "status failed")
		return
	}
	listen := ""
	active := false
	startErr := ""
	traffic := metrics.Snapshot{}
	if h.proxy != nil {
		active = h.proxy.ProxyActive()
		listen = h.proxy.ProxyListen()
		startErr = h.proxy.ProxyLastStartError()
		traffic = h.proxy.ProxyTraffic()
	}
	if listen == "" {
		if st, err := h.settings.Get(r.Context()); err == nil {
			listen = st.Listen
		}
	}
	if active {
		startErr = ""
	}
	response.JSON(w, http.StatusOK, proxyStatusResponse{
		ProxyActive:     active,
		Listen:          listen,
		ProxyStartError: startErr,
		ACL:             aclSt,
		Traffic:         traffic,
		Startup:         startup.Timings(),
	})
}

func (h *StatusHandler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return h.auth.Require(next)
}

type proxyStatusResponse struct {
	ProxyActive     bool                             `json:"proxy_active"`
	Listen          string                           `json:"listen"`
	ProxyStartError string                           `json:"proxy_start_error,omitempty"`
	ACL             proxyaclservice.CompileStatusDTO `json:"acl"`
	Traffic         metrics.Snapshot                 `json:"traffic"`
	Startup         startup.TimingsDTO               `json:"startup"`
}
