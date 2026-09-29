package squid_test

import (
	"testing"

	"oktopus/internal/proxy/acl/squid"
)

func TestListNamesToMergeFromDB(t *testing.T) {
	cfg := `
acl localnet src 10.0.0.0/8
http_access deny blocked
http_access allow localnet
`
	parsed, _ := squid.ParsePolicyConfig(cfg)
	if parsed == nil {
		t.Fatal("parse")
	}
	names := squid.ListNamesToMergeFromDB(parsed)
	if len(names) != 1 || names[0] != "blocked" {
		t.Fatalf("names=%v want [blocked]", names)
	}
}
