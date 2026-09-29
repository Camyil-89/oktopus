package squid_test

import (
	"testing"

	"oktopus/internal/proxy/acl/squid"
)

func TestFormatAccessDirective_httpAccess(t *testing.T) {
	got := squid.FormatAccessDirective("http_access", "allow", []squid.AccessClause{
		{Name: "all"},
	})
	if got != "http_access allow all" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatAccessDirective_negated(t *testing.T) {
	got := squid.FormatAccessDirective("http_access", "deny", []squid.AccessClause{
		{Name: "localnet", Negated: true},
		{Name: "all"},
	})
	if got != "http_access deny !localnet all" {
		t.Fatalf("got %q", got)
	}
}
