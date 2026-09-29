package squid_test

import (
	"testing"

	"oktopus/internal/proxy/acl/squid"
)

func TestCommentLinesIgnored(t *testing.T) {
	cfg := `
# acl и http_access (как в Squid)
acl localnet src 127.0.0.1
acl all all
# http_access deny localnet
http_access allow all
`
	res := squid.Analyze(cfg, nil)
	if !res.OK {
		t.Fatalf("expected ok, got %v", res.Diagnostics)
	}
}

func TestBOMCommentLine(t *testing.T) {
	cfg := "\ufeff# comment\nacl all all\nhttp_access allow all\n"
	res := squid.Analyze(cfg, nil)
	if !res.OK {
		t.Fatalf("expected ok, got %v", res.Diagnostics)
	}
}
