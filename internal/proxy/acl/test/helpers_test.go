package acl_test

import (
	"fmt"
	"strings"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/acl/squid"
)

const benchRuleCount = 10_000

func benchHostLines(n int) string {
	var lines strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			lines.WriteByte('\n')
		}
		fmt.Fprintf(&lines, "host%d.example.com", i)
	}
	return lines.String()
}

// Один dstdomain list + одна строка http_access deny.
func benchSquidOneListDeny(n int) (*acl.Engine, error) {
	return squid.Compile(`
http_access deny deny_hosts
`, []squid.ListInput{
		{Name: "deny_hosts", ListType: "dstdomain", Body: benchHostLines(n)},
	})
}

// N отдельных acl dstdomain + N строк http_access deny.
func benchSquidSeparateDenyRules(n int) (*acl.Engine, error) {
	var cfg strings.Builder
	cfg.WriteString("acl all all\n")
	for i := 0; i < n; i++ {
		fmt.Fprintf(&cfg, "acl h%d dstdomain host%d.example.com\n", i, i)
		fmt.Fprintf(&cfg, "http_access deny h%d\n", i)
	}
	return squid.Compile(cfg.String(), nil)
}

func benchSquidDevLike(n int) (*acl.Engine, error) {
	var body strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			body.WriteByte('\n')
		}
		fmt.Fprintf(&body, "blocked%d.evil", i)
	}
	return squid.Compile(`
acl allow_grp ldap_group allow_all
acl all all
http_access deny blocked
http_access allow allow_grp
http_access allow all
`, []squid.ListInput{
		{Name: "blocked", ListType: "dstdomain", Body: body.String()},
	})
}

func mustEngineFromSquid(t testingT, config string, lists ...squid.ListInput) *acl.Engine {
	t.Helper()
	e, err := squid.Compile(config, lists)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

type testingT interface {
	Helper()
	Fatal(args ...any)
}
