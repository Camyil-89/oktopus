package squid_test

import (
	"fmt"
	"strings"
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/acl/squid"
	"oktopus/internal/proxy/auth"
)

func BenchmarkSquidDstDomain300k_NoMatch(b *testing.B) {
	var body strings.Builder
	for i := 0; i < 300_000; i++ {
		body.WriteString(fmt.Sprintf("host%d.blocked.test\n", i))
	}
	engine, err := squid.Compile(`
acl all all
http_access deny blocked
http_access allow all
`, []squid.ListInput{
		{Name: "blocked", ListType: "dstdomain", Body: body.String()},
	})
	if err != nil {
		b.Fatal(err)
	}
	st := engine.SNIPatternStats()
	if st.Indexed < 300_000 {
		b.Fatalf("expected indexed domains, got indexed=%d regexp=%d", st.Indexed, st.Regexp)
	}
	id := auth.Identity{}
	f := acl.RequestFields{SNI: "safe.example.com", DstPort: 443}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !engine.Decide(id, f) {
			b.Fatal("expected allow")
		}
	}
}
