package acl_test

import (
	"testing"

	"oktopus/internal/proxy/acl"
	"oktopus/internal/proxy/acl/squid"
)

func TestSNINormalizeAdblockLineToIndex(t *testing.T) {
	e := mustEngineFromSquid(t, `
http_access deny bad
`, squid.ListInput{Name: "bad", ListType: "dstdomain", Body: "||blocked.evil^"})
	st := e.SNIPatternStats()
	if st.Indexed != 1 || st.Regexp != 0 {
		t.Fatalf("stats indexed=%d regexp=%d", st.Indexed, st.Regexp)
	}
}

func TestClassifySNIPatternIndex(t *testing.T) {
	ok, reason := acl.ClassifySNIPatternIndex("||foo.example.com^")
	if !ok || reason != "" {
		t.Fatalf("want indexed, got ok=%v reason=%q", ok, reason)
	}
	ok, reason = acl.ClassifySNIPatternIndex("(?i)blocked\\.evil")
	if !ok || reason != "" {
		t.Fatalf("simple (?i) host should index, got ok=%v reason=%q", ok, reason)
	}
	ok, reason = acl.ClassifySNIPatternIndex("^.*\\.evil$")
	if ok || reason != "regex_syntax" {
		t.Fatalf("want regex_syntax, got ok=%v reason=%q", ok, reason)
	}
}
