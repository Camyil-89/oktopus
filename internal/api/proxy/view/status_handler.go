package view

import (
	"net/http"

	"github.com/google/uuid"

	authmw "oktopus/internal/api/auth/middleware"
	"oktopus/internal/api/platform/response"
	proxyaclservice "oktopus/internal/db/proxyacl/service"
	proxyinstancesservice "oktopus/internal/db/proxyinstances/service"
	"oktopus/internal/proxy/metrics"
	"oktopus/internal/proxy/server"
	"oktopus/internal/startup"
)

type FleetStatusProvider interface {
	InstanceStatuses() []server.InstanceStatus
	AggregateTraffic() metrics.Snapshot
	ProxyActive() bool
}

type StatusHandler struct {
	instances *proxyinstancesservice.Service
	acl       *proxyaclservice.Service
	fleet     FleetStatusProvider
	auth      *authmw.Guard
}

func NewStatusHandler(
	instances *proxyinstancesservice.Service,
	acl *proxyaclservice.Service,
	fleet FleetStatusProvider,
	auth *authmw.Guard,
) *StatusHandler {
	return &StatusHandler{instances: instances, acl: acl, fleet: fleet, auth: auth}
}

func (h *StatusHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	fleetByID := map[uuid.UUID]server.InstanceStatus{}
	var instances []instanceStatusDTO
	aggregate := metrics.Snapshot{}
	active := false
	if h.fleet != nil {
		for _, st := range h.fleet.InstanceStatuses() {
			fleetByID[st.InstanceID] = st
		}
		aggregate = h.fleet.AggregateTraffic()
		active = h.fleet.ProxyActive()
	}
	dbList, err := h.instances.List(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "load instances failed")
		return
	}
	for _, inst := range dbList {
		st := fleetByID[inst.ID]
		aclSt, _ := h.acl.CompileStatus(r.Context(), inst.ID)
		startErr := st.LastStartError
		if st.Active {
			startErr = ""
		}
		listen := inst.Listen
		if st.Listen != "" {
			listen = st.Listen
		}
		instances = append(instances, instanceStatusDTO{
			ID:              inst.ID.String(),
			Listen:          listen,
			Active:          st.Active,
			ProxyStartError: startErr,
			ACL:             aclSt,
			Traffic:         st.Traffic,
		})
	}
	response.JSON(w, http.StatusOK, proxyStatusResponse{
		ProxyActive: active,
		Traffic:     aggregate,
		Instances:   instances,
		Startup:     startup.Timings(),
	})
}

func (h *StatusHandler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return h.auth.Require(next)
}

type instanceStatusDTO struct {
	ID              string                           `json:"id"`
	Listen          string                           `json:"listen"`
	Active          bool                             `json:"active"`
	ProxyStartError string                           `json:"proxy_start_error,omitempty"`
	ACL             proxyaclservice.CompileStatusDTO `json:"acl"`
	Traffic         metrics.Snapshot                 `json:"traffic"`
}

type proxyStatusResponse struct {
	ProxyActive bool                `json:"proxy_active"`
	Traffic     metrics.Snapshot    `json:"traffic"`
	Instances   []instanceStatusDTO `json:"instances"`
	Startup     startup.TimingsDTO  `json:"startup"`
}
