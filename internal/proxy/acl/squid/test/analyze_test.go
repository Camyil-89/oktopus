package squid_test

import (
	"testing"

	"oktopus/internal/proxy/acl/squid"
)

func TestAnalyzeUnknownACL(t *testing.T) {
	cfg := `
acl all all
http_access deny missing_acl
`
	res := squid.Analyze(cfg, nil)
	if res.OK {
		t.Fatal("expected failure")
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "unknown_acl" && d.ACLName == "missing_acl" {
			found = true
			if d.Line != 3 {
				t.Fatalf("line %d", d.Line)
			}
			if d.Column < 1 {
				t.Fatalf("column %d", d.Column)
			}
		}
	}
	if !found {
		t.Fatalf("diagnostics: %v", res.Diagnostics)
	}
}
