package service

import (
	"encoding/json"
	"net/url"
	"strings"

	"oktopus/internal/proxy/accesslog"
	proxyhttp "oktopus/internal/proxy/http"
)

func buildExtraJSON(e accesslog.Entry) []byte {
	m := make(map[string]any)
	if e.InspectError != "" {
		m["inspect_error"] = e.InspectError
	}
	if typ := strings.TrimSpace(e.GatewayErrorType); typ != "" {
		ge := map[string]any{"type": typ}
		if e.InspectError != "" {
			ge["message"] = e.InspectError
		}
		m["gateway_error"] = ge
	}
	fullURL := strings.TrimSpace(e.FullURLRequest)
	if fullURL != "" {
		if u, err := url.Parse(fullURL); err == nil {
			engine, query := proxyhttp.ParseSearchRequest(u)
			if engine != "" && query != "" {
				m["search"] = map[string]any{
					"engine": engine,
					"query":  query,
				}
			}
		}
	}
	for ruleID, payload := range e.InspectRuleLogs {
		if ruleID == "" || payload == nil {
			continue
		}
		m[ruleID] = payload
	}
	if len(m) == 0 {
		return []byte("{}")
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return []byte("{}")
	}
	return raw
}
