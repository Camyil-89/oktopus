package view

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	authmw "oktopus/internal/api/auth/middleware"
	"oktopus/internal/api/platform/response"
	"oktopus/internal/db/proxyaccesslog/domain"
	"oktopus/internal/db/proxyaccesslog/repository"
	"oktopus/internal/proxy/observe"
	proxyaccesslogservice "oktopus/internal/db/proxyaccesslog/service"
)

type AccessLogHandler struct {
	logs *proxyaccesslogservice.Service
	auth *authmw.Guard
}

func NewAccessLogHandler(logs *proxyaccesslogservice.Service, auth *authmw.Guard) *AccessLogHandler {
	return &AccessLogHandler{logs: logs, auth: auth}
}

func (h *AccessLogHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	filter := repository.ListFilter{
		InstanceID:      strings.TrimSpace(r.URL.Query().Get("instance_id")),
		ID:              strings.TrimSpace(r.URL.Query().Get("id")),
		User:            strings.TrimSpace(r.URL.Query().Get("user")),
		Source:          strings.TrimSpace(r.URL.Query().Get("source")),
		Destination:     strings.TrimSpace(r.URL.Query().Get("destination")),
		URL:             strings.TrimSpace(r.URL.Query().Get("url")),
		SearchOnly:      queryBool(r.URL.Query().Get("search_only")),
		ErrorKind:       parseAccessLogErrorKind(r.URL.Query().Get("error_kind")),
		Segment:         strings.TrimSpace(r.URL.Query().Get("segment")),
		AttackKind:      "",
		PolicyAnomalyQ:  strings.TrimSpace(r.URL.Query().Get("policy_anomaly_q")),
		DecisionRuleRef: strings.TrimSpace(r.URL.Query().Get("decision_rule_ref")),
		InspectRuleID:   strings.TrimSpace(r.URL.Query().Get("inspect_rule_id")),
	}
	if !repository.ValidAccessLogSegment(filter.Segment) {
		response.Error(w, http.StatusBadRequest, "invalid segment")
		return
	}
	if kind, err := parseAccessLogAttackKind(r.URL.Query().Get("attack_kind")); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid attack_kind")
		return
	} else {
		filter.AttackKind = kind
	}
	if filter.ID != "" {
		if _, err := uuid.Parse(filter.ID); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid id")
			return
		}
	}
	if actionRaw := strings.TrimSpace(r.URL.Query().Get("action")); actionRaw != "" {
		action, err := strconv.ParseInt(actionRaw, 10, 32)
		if err != nil || (action != 0 && action != 1) {
			response.Error(w, http.StatusBadRequest, "invalid action")
			return
		}
		filter.Action = int32(action)
	} else {
		filter.Action = -1
	}
	if filter.InspectRuleID != "" {
		if _, err := uuid.Parse(filter.InspectRuleID); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid inspect_rule_id")
			return
		}
	}
	if filter.InstanceID != "" {
		if _, err := uuid.Parse(filter.InstanceID); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid instance_id")
			return
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("from")); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid from")
			return
		}
		filter.From = &t
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("to")); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "invalid to")
			return
		}
		filter.To = &t
	}

	pageData, err := h.logs.List(r.Context(), filter, page, pageSize)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "list access log failed")
		return
	}

	results := make([]accessLogRowResponse, 0, len(pageData.Items))
	for _, row := range pageData.Items {
		results = append(results, toAccessLogRow(row))
	}

	response.JSON(w, http.StatusOK, accessLogListResponse{
		Count:    pageData.Total,
		Page:     pageData.Page,
		PageSize: pageData.Size,
		Results:  results,
	})
}

func (h *AccessLogHandler) DeleteByPeriod(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var body deleteByPeriodRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	from, err := time.Parse(time.RFC3339, strings.TrimSpace(body.From))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid from")
		return
	}
	to, err := time.Parse(time.RFC3339, strings.TrimSpace(body.To))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid to")
		return
	}

	deleted, err := h.logs.DeleteBetween(r.Context(), from, to)
	if err != nil {
		if strings.Contains(err.Error(), "period") {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, "delete access log failed")
		return
	}

	response.JSON(w, http.StatusOK, deleteByPeriodResponse{Deleted: deleted})
}

func queryBool(raw string) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	return raw == "1" || raw == "true" || raw == "yes"
}

func parseAccessLogErrorKind(raw string) string {
	if strings.TrimSpace(strings.ToLower(raw)) == "any" {
		return "any"
	}
	return ""
}

func parseAccessLogAttackKind(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if raw == observe.PolicyAnomalyHostSNIMismatch {
		return raw, nil
	}
	if len(raw) > 64 {
		return "", fmt.Errorf("too long")
	}
	for _, c := range raw {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_' {
			continue
		}
		return "", fmt.Errorf("bad char")
	}
	return raw, nil
}

func (h *AccessLogHandler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return h.auth.Require(next)
}

type deleteByPeriodRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type deleteByPeriodResponse struct {
	Deleted int64 `json:"deleted"`
}

type accessLogListResponse struct {
	Count    int64                  `json:"count"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Results  []accessLogRowResponse `json:"results"`
}

type accessLogRowResponse struct {
	ID                 string          `json:"id"`
	InstanceID         string          `json:"instance_id"`
	CreatedAt          string          `json:"created_at"`
	SourceAddress      string          `json:"source_address"`
	DestinationAddress string          `json:"destination_address"`
	User               *string         `json:"user"`
	DecideDurationUs   int64           `json:"decide_duration_us"`
	FullURL            string          `json:"full_url"`
	Action             int16           `json:"action"`
	InspectRuleID      *string         `json:"inspect_rule_id"`
	DeniedBy           *string         `json:"denied_by"`
	DecisionRuleRef    string          `json:"decision_rule_ref,omitempty"`
	Extra              json.RawMessage `json:"extra"`
}

func toAccessLogRow(row domain.Entry) accessLogRowResponse {
	extra := row.Extra
	if len(extra) == 0 {
		extra = []byte("{}")
	}
	var inspectRuleID *string
	if row.InspectRuleID != nil {
		s := row.InspectRuleID.String()
		inspectRuleID = &s
	}
	var deniedBy *string
	if row.DeniedBy != nil && *row.DeniedBy != "" {
		deniedBy = row.DeniedBy
	}
	return accessLogRowResponse{
		ID:                 row.ID.String(),
		InstanceID:         row.InstanceID.String(),
		CreatedAt:          row.CreatedAt.UTC().Format(time.RFC3339),
		SourceAddress:      row.SourceAddress,
		DestinationAddress: row.DestinationAddress,
		User:               row.UserName,
		DecideDurationUs:   row.DecideDurationUs,
		FullURL:            row.FullURL,
		Action:             row.Action,
		InspectRuleID:      inspectRuleID,
		DeniedBy:           deniedBy,
		DecisionRuleRef:    row.DecisionRuleRef,
		Extra:              json.RawMessage(extra),
	}
}
