package view

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	authmw "oktopus/internal/api/auth/middleware"
	"oktopus/internal/api/platform/response"
	"oktopus/internal/db/proxyacl/domain"
	proxyaclservice "oktopus/internal/db/proxyacl/service"
	"oktopus/internal/proxy/acl/squid"

	"github.com/google/uuid"
)

type ACLHandler struct {
	acl  *proxyaclservice.Service
	auth *authmw.Guard
}

func NewACLHandler(acl *proxyaclservice.Service, auth *authmw.Guard) *ACLHandler {
	return &ACLHandler{acl: acl, auth: auth}
}

func (h *ACLHandler) Status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	st, err := h.acl.CompileStatus(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "acl status failed")
		return
	}
	response.JSON(w, http.StatusOK, st)
}

func (h *ACLHandler) Evaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body evaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	result := h.acl.Evaluate(r.Context(), proxyaclservice.EvaluateInput{
		SNI:      body.SNI,
		Path:     body.Path,
		SrcIP:    body.SrcIP,
		DstIP:    body.DstIP,
		DstPort:  body.DstPort,
		Username: body.Username,
		Groups:   body.Groups,
	})
	response.JSON(w, http.StatusOK, result)
}

type evaluateRequest struct {
	SNI      string   `json:"sni"`
	Path     string   `json:"path"`
	SrcIP    string   `json:"src_ip"`
	DstIP    string   `json:"dst_ip"`
	DstPort  int      `json:"dst_port"`
	Username string   `json:"username"`
	Groups   []string `json:"groups"`
}

func (h *ACLHandler) ValidatePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body validatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	var listInputs []squid.ListInput
	if body.Lists != nil {
		listInputs = make([]squid.ListInput, len(*body.Lists))
		for i, l := range *body.Lists {
			listInputs[i] = squid.ListInput{
				Name:     l.Name,
				ListType: l.ListType,
				Body:     l.Body,
			}
		}
	}
	result := h.acl.AnalyzePolicy(r.Context(), body.ConfigText, listInputs)
	response.JSON(w, http.StatusOK, result)
}

func (h *ACLHandler) GetPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	pol, err := h.acl.GetPolicy(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "get acl policy failed")
		return
	}
	response.JSON(w, http.StatusOK, aclPolicyResponse{
		ConfigText: pol.ConfigText,
		UpdatedAt:  pol.UpdatedAt,
	})
}

func (h *ACLHandler) PutPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body putPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	pol, err := h.acl.SetPolicyAndPublish(r.Context(), body.ConfigText)
	if err != nil {
		response.ErrorWithDiagnostics(w, http.StatusBadRequest, err.Error(), proxyaclservice.DiagnosticsFromError(err))
		return
	}
	response.JSON(w, http.StatusOK, aclPolicyResponse{
		ConfigText: pol.ConfigText,
		UpdatedAt:  pol.UpdatedAt,
	})
}

func (h *ACLHandler) ListNamedLists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	lists, err := h.acl.ListNamedListsSummary(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "list acl lists failed")
		return
	}
	out := make([]aclNamedListSummaryResponse, len(lists))
	for i, l := range lists {
		out[i] = toACLNamedListSummaryResponse(l)
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *ACLHandler) GetNamedList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	listID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid list id")
		return
	}
	list, err := h.acl.GetNamedList(r.Context(), listID)
	if err != nil {
		if errors.Is(err, domain.ErrListNotFound) {
			response.Error(w, http.StatusNotFound, "list not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "get acl list failed")
		return
	}
	response.JSON(w, http.StatusOK, toACLNamedListDetailResponse(list))
}

func (h *ACLHandler) PollNamedList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	listID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid list id")
		return
	}
	var body pollNamedListRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid json")
			return
		}
	}
	list, err := h.acl.PollNamedListNow(r.Context(), listID, body.SourceURL)
	if err != nil {
		if errors.Is(err, domain.ErrListNotFound) {
			response.Error(w, http.StatusNotFound, "list not found")
			return
		}
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, toACLNamedListDetailResponse(list))
}

func (h *ACLHandler) SyncNamedLists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body syncNamedListsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	inputs := make([]proxyaclservice.SyncNamedListInput, len(body.Lists))
	for i, l := range body.Lists {
		inputs[i] = proxyaclservice.SyncNamedListInput{
			ID:                  l.ID,
			Name:                l.Name,
			ListType:            l.ListType,
			Body:                l.Body,
			SourceMode:          l.SourceMode,
			SourceURL:           l.SourceURL,
			PollIntervalMinutes: l.PollIntervalMinutes,
		}
	}
	if _, err := h.acl.SyncNamedListsAndPublish(r.Context(), inputs); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	summaries, err := h.acl.ListNamedListsSummary(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "list acl lists failed")
		return
	}
	out := make([]aclNamedListSummaryResponse, len(summaries))
	for i, l := range summaries {
		out[i] = toACLNamedListSummaryResponse(l)
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *ACLHandler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return h.auth.Require(next)
}

type aclPolicyResponse struct {
	ConfigText string    `json:"config_text"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type putPolicyRequest struct {
	ConfigText string `json:"config_text"`
}

type validatePolicyRequest struct {
	ConfigText string                   `json:"config_text"`
	Lists      *[]aclNamedListSyncItem  `json:"lists"`
}

type aclNamedListSummaryResponse struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	ListType            string    `json:"list_type"`
	SourceMode          string    `json:"source_mode"`
	SourceURL           string    `json:"source_url"`
	PollIntervalMinutes int       `json:"poll_interval_minutes"`
	BodyLineCount       int       `json:"body_line_count"`
	BodyPreview         string    `json:"body_preview"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type aclNamedListDetailResponse struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	ListType            string    `json:"list_type"`
	Body                string    `json:"body"`
	SourceMode          string    `json:"source_mode"`
	SourceURL           string    `json:"source_url"`
	PollIntervalMinutes int       `json:"poll_interval_minutes"`
	BodyLineCount       int       `json:"body_line_count"`
	BodyPreview         string    `json:"body_preview"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type syncNamedListsRequest struct {
	Lists []aclNamedListSyncItem `json:"lists"`
}

type pollNamedListRequest struct {
	SourceURL string `json:"source_url"`
}

type aclNamedListSyncItem struct {
	ID                  *string `json:"id"`
	Name                string  `json:"name"`
	ListType            string  `json:"list_type"`
	Body                string  `json:"body"`
	SourceMode          string  `json:"source_mode"`
	SourceURL           string  `json:"source_url"`
	PollIntervalMinutes int     `json:"poll_interval_minutes"`
}

func toACLNamedListSummaryResponse(l domain.NamedListSummary) aclNamedListSummaryResponse {
	return aclNamedListSummaryResponse{
		ID:                  l.ID.String(),
		Name:                l.Name,
		ListType:            l.ListType,
		SourceMode:          l.SourceMode,
		SourceURL:           l.SourceURL,
		PollIntervalMinutes: l.PollIntervalMinutes,
		BodyLineCount:       l.BodyLineCount,
		BodyPreview:         l.BodyPreview,
		CreatedAt:           l.CreatedAt,
		UpdatedAt:           l.UpdatedAt,
	}
}

func toACLNamedListDetailResponse(l domain.NamedList) aclNamedListDetailResponse {
	lineCount, preview := domain.NamedListBodyStats(l.Body)
	return aclNamedListDetailResponse{
		ID:                  l.ID.String(),
		Name:                l.Name,
		ListType:            l.ListType,
		Body:                l.Body,
		SourceMode:          l.SourceMode,
		SourceURL:           l.SourceURL,
		PollIntervalMinutes: l.PollIntervalMinutes,
		BodyLineCount:       lineCount,
		BodyPreview:         preview,
		CreatedAt:           l.CreatedAt,
		UpdatedAt:           l.UpdatedAt,
	}
}
