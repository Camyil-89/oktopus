package service

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"oktopus/internal/db/proxyaccesslog/extra"
	"oktopus/internal/proxy/accesslog"
	"oktopus/internal/proxy/observe"
)

func Test_buildExtraJSON_google(t *testing.T) {
	raw := buildExtraJSON(accesslog.Entry{FullURLRequest: "https://www.google.com/search?q=hello+world"})
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	search, _ := out["search"].(map[string]any)
	if search == nil || search["query"] != "hello world" {
		t.Fatalf("query: %+v", search)
	}
}

func Test_buildExtraJSON_noSearch(t *testing.T) {
	raw := buildExtraJSON(accesslog.Entry{FullURLRequest: "https://example.com/page"})
	if string(raw) != "{}" {
		t.Fatalf("got %s", raw)
	}
}

func Test_buildExtraJSON_googleNonSearchPath(t *testing.T) {
	raw := buildExtraJSON(accesslog.Entry{FullURLRequest: "https://www.googleapis.com/oauth2/v4/token?p=secret"})
	if string(raw) != "{}" {
		t.Fatalf("got %s", raw)
	}
}

func Test_fromAccessEntry_ruleColumns(t *testing.T) {
	aclID := uuid.MustParse("018f0000-0000-7000-8000-000000000001")
	row := fromAccessEntry(accesslog.Entry{
		RuleRef:    aclID.String(),
		ACLRuleRef: aclID.String(),
		DeniedBy:   accesslog.DeniedByACL,
		Action:     accesslog.ActionDeny,
	})
	if row.DecisionRuleRef != aclID.String() {
		t.Fatalf("decision: %q", row.DecisionRuleRef)
	}
	if row.InspectRuleID != nil {
		t.Fatalf("inspect should be nil for acl-only entry")
	}
	if row.DeniedBy == nil || *row.DeniedBy != accesslog.DeniedByACL {
		t.Fatalf("denied_by: %v", row.DeniedBy)
	}
}

func Test_buildExtraJSON_policyAnomaly_roundTrip(t *testing.T) {
	raw := buildExtraJSON(accesslog.Entry{
		PolicyNameMismatch: &observe.PolicyNameMismatch{
			Kind:         observe.PolicyAnomalyHostSNIMismatch,
			ConnectHost:  "localhost",
			TLSClientSNI: "internal.blocked",
			PolicyHost:   "localhost",
		},
	})
	parsed := extra.ParseFromJSON(uuid.Nil, raw)
	if len(parsed.KV) != 1 || parsed.KV[0].FieldKey != "policy_anomaly" {
		t.Fatalf("kv: %+v", parsed.KV)
	}
	merged := extra.BuildJSON("", "", "", parsed.KV)
	var out map[string]json.RawMessage
	if err := json.Unmarshal(merged, &out); err != nil {
		t.Fatal(err)
	}
	if out["policy_anomaly"] == nil {
		t.Fatalf("missing policy_anomaly: %s", merged)
	}
}

func Test_buildExtraJSON_inspectRuleLog(t *testing.T) {
	ruleID := "018f0000-0000-7000-8000-000000000001"
	raw := buildExtraJSON(accesslog.Entry{
		InspectRuleLogs: map[string]map[string]any{
			ruleID: {"filenames": []any{"a.pdf"}},
		},
	})
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	block, ok := out[ruleID].(map[string]any)
	if !ok {
		t.Fatalf("missing rule block: %v", out)
	}
	if block["filenames"] == nil {
		t.Fatalf("filenames: %v", block)
	}
}
