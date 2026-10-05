package view

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	authmw "oktopus/internal/api/auth/middleware"
	"oktopus/internal/api/platform/response"
	"oktopus/internal/db/proxyinspect/domain"
	proxyinspectservice "oktopus/internal/db/proxyinspect/service"

	"github.com/google/uuid"
)

type InspectHandler struct {
	inspect *proxyinspectservice.Service
	auth    *authmw.Guard
}

func NewInspectHandler(inspect *proxyinspectservice.Service, auth *authmw.Guard) *InspectHandler {
	return &InspectHandler{inspect: inspect, auth: auth}
}

func (h *InspectHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	instanceID, ok := parseInstanceID(w, r)
	if !ok {
		return
	}
	rules, err := h.inspect.List(r.Context(), instanceID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "list inspect rules failed")
		return
	}
	out := make([]inspectRuleResponse, len(rules))
	for i := range rules {
		out[i] = toInspectRuleResponse(rules[i])
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *InspectHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	ruleID, err := uuid.Parse(r.PathValue("ruleId"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	rule, err := h.inspect.GetRule(r.Context(), ruleID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			response.Error(w, http.StatusNotFound, "rule not found")
			return
		}
		response.Error(w, http.StatusInternalServerError, "get inspect rule failed")
		return
	}
	response.JSON(w, http.StatusOK, toInspectRuleResponse(rule))
}

func (h *InspectHandler) SyncRules(w http.ResponseWriter, r *http.Request) {
	var body syncInspectRulesRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	inputs := make([]proxyinspectservice.SyncRuleInput, len(body.Rules))
	for i, rule := range body.Rules {
		inputs[i] = proxyinspectservice.SyncRuleInput{
			ID:        rule.ID,
			Name:      rule.Name,
			Script:    rule.Script,
			Action:    rule.Action,
			Enabled:   rule.Enabled,
			SortOrder: rule.SortOrder,
		}
	}
	instanceID, ok := parseInstanceID(w, r)
	if !ok {
		return
	}
	rules, err := h.inspect.SyncAndPublish(r.Context(), instanceID, inputs)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	out := make([]inspectRuleListItemResponse, len(rules))
	for i := range rules {
		out[i] = toInspectRuleListItemResponse(rules[i])
	}
	response.JSON(w, http.StatusOK, out)
}

func (h *InspectHandler) Status(w http.ResponseWriter, r *http.Request) {
	instanceID, ok := parseInstanceID(w, r)
	if !ok {
		return
	}
	st, err := h.inspect.CompileStatus(r.Context(), instanceID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "inspect status failed")
		return
	}
	response.JSON(w, http.StatusOK, st)
}

func (h *InspectHandler) ValidateScript(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body validateInspectScriptRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid json")
		return
	}
	if err := h.inspect.ValidateScript(r.Context(), body.Script); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, validateInspectScriptResponse{OK: true})
}

func (h *InspectHandler) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return h.auth.Require(next)
}

type inspectRuleListItemResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Action       int16  `json:"action"`
	Enabled      bool   `json:"enabled"`
	SortOrder    int    `json:"sort_order"`
	ScriptLength int    `json:"script_length"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type inspectRuleResponse struct {
	inspectRuleListItemResponse
	Script string `json:"script"`
}

type syncInspectRulePayload struct {
	ID        *string `json:"id"`
	Name      string  `json:"name"`
	Script    *string `json:"script"`
	Action    int16   `json:"action"`
	Enabled   bool    `json:"enabled"`
	SortOrder int     `json:"sort_order"`
}

type syncInspectRulesRequest struct {
	Rules []syncInspectRulePayload `json:"rules"`
}

type validateInspectScriptRequest struct {
	Script string `json:"script"`
}

type validateInspectScriptResponse struct {
	OK bool `json:"ok"`
}

func toInspectRuleListItemResponse(r domain.RuleSummary) inspectRuleListItemResponse {
	return inspectRuleListItemResponse{
		ID:           r.ID.String(),
		Name:         r.Name,
		Action:       r.Action,
		Enabled:      r.Enabled,
		SortOrder:    r.SortOrder,
		ScriptLength: r.ScriptLength,
		CreatedAt:    r.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    r.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toInspectRuleResponse(r domain.Rule) inspectRuleResponse {
	return inspectRuleResponse{
		inspectRuleListItemResponse: toInspectRuleListItemResponse(domain.RuleSummary{
			ID:           r.ID,
			Name:         r.Name,
			Action:       r.Action,
			Enabled:      r.Enabled,
			SortOrder:    r.SortOrder,
			ScriptLength: len(r.Script),
			CreatedAt:    r.CreatedAt,
			UpdatedAt:    r.UpdatedAt,
		}),
		Script: r.Script,
	}
}
