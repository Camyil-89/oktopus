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

func sniListBody(n int, extraLine string) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "host%d.example.com", i)
	}
	if extraLine != "" {
		b.WriteByte('\n')
		b.WriteString(extraLine)
	}
	return b.String()
}

func compileSNIListEngine(n int, extraLine string) (*acl.Engine, error) {
	return squid.Compile(`
http_access deny deny_hosts
`, []squid.ListInput{
		{Name: "deny_hosts", ListType: "dstdomain", Body: sniListBody(n, extraLine)},
	})
}

func TestLargeRegexsLiteralOnlyNotSlowSlot(t *testing.T) {
	const n = 8000
	e, err := compileSNIListEngine(n, "")
	if err != nil {
		t.Fatal(err)
	}
	if e.SlowRuleSlotCount() != 0 {
		t.Fatalf("literal-only list must not use slow rule slots, got %d", e.SlowRuleSlotCount())
	}
	st := e.SNIPatternStats()
	if st.Indexed != n || st.Regexp != 0 {
		t.Fatalf("sni stats: indexed=%d regexp=%d want %d/0", st.Indexed, st.Regexp, n)
	}
}

func TestLargeRegexsWithOneRegexpLineNoLinearScan(t *testing.T) {
	const n = 8000
	e, err := compileSNIListEngine(n, "^bad\\..+")
	if err != nil {
		t.Fatal(err)
	}
	if e.SlowRuleSlotCount() != 1 {
		t.Fatalf("expected 1 slow slot, got %d", e.SlowRuleSlotCount())
	}
	st := e.SNIPatternStats()
	if st.Indexed != n || st.Regexp != 1 {
		t.Fatalf("sni stats: indexed=%d regexp=%d", st.Indexed, st.Regexp)
	}
	id := auth.Identity{Username: "u"}
	start := time.Now()
	for i := 0; i < 500; i++ {
		if e.Decide(id, acl.RequestFields{SNI: "ok.example.com", Path: "/"}) {
			t.Fatal("default deny")
		}
	}
	if d := time.Since(start); d > 250*time.Millisecond {
		t.Fatalf("500 decides took %v; expected index path, not O(n) scan", d)
	}
}
