package acl_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/acl/squid"
	"oktopus/internal/proxy/auth"
)

func sniManyRegexpLinesBody(literals, regexLines int) string {
	var b strings.Builder
	for i := 0; i < literals; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "host%d.example.com", i)
	}
	for i := 0; i < regexLines; i++ {
		b.WriteByte('\n')
		fmt.Fprintf(&b, "^bad%d\\..+", i)
	}
	return b.String()
}

func TestSlowRuleManyRegexpOneMatchPass(t *testing.T) {
	const literals = 4000
	const regexLines = 40
	e, err := squid.Compile(`
http_access deny deny_hosts
`, []squid.ListInput{
		{Name: "deny_hosts", ListType: "dstdomain", Body: sniManyRegexpLinesBody(literals, regexLines)},
	})
	if err != nil {
		t.Fatal(err)
	}
	st := e.SNIPatternStats()
	if st.Indexed != literals || st.Regexp != regexLines {
		t.Fatalf("indexed=%d regexp=%d", st.Indexed, st.Regexp)
	}
	if e.SlowRuleSlotCount() != 1 {
		t.Fatalf("slow slots=%d", e.SlowRuleSlotCount())
	}
	id := auth.Identity{Username: "u"}
	start := time.Now()
	for i := 0; i < 400; i++ {
		if e.Decide(id, acl.RequestFields{SNI: "ok.example.com", Path: "/"}) {
			t.Fatal("default deny")
		}
	}
	if d := time.Since(start); d > 200*time.Millisecond {
		t.Fatalf("400 decides took %v", d)
	}
}
