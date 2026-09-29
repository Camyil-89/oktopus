package extra

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestParseAndBuildRoundTrip(t *testing.T) {
	logID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	ruleID := "22222222-2222-4222-8222-222222222222"
	raw := []byte(`{
		"search":{"engine":"google","query":"cats"},
		"inspect_error":"boom",
		"` + ruleID + `":{"upload_method":"POST","upload_filenames":["a.pdf"]}
	}`)
	parsed := ParseFromJSON(logID, raw)
	if parsed.SearchEngine != "google" || parsed.SearchQuery != "cats" {
		t.Fatalf("search: %+v", parsed)
	}
	if parsed.InspectError != "boom" {
		t.Fatalf("inspect_error")
	}
	if len(parsed.KV) != 2 {
		t.Fatalf("kv len %d", len(parsed.KV))
	}
	out := BuildJSON(parsed.SearchEngine, parsed.SearchQuery, parsed.InspectError, parsed.KV)
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m[ruleID]; !ok {
		t.Fatalf("missing rule block: %s", out)
	}
}
