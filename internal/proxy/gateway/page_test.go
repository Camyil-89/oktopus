package gateway

import (
	"regexp"
	"strings"
	"testing"
)

func TestRender502SubstitutesError(t *testing.T) {
	_ = Reload()
	body := string(Render502(errSample("tls: bad cert")))
	if strings.Contains(body, "tls: bad cert") {
		t.Fatalf("upstream text must not appear on page: %s", body)
	}
	if strings.Contains(body, "{{ERROR}}") || strings.Contains(body, "{{ERROR_ID}}") {
		t.Fatal("placeholder not replaced")
	}
	if !regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`).MatchString(body) {
		t.Fatalf("expected incident uuid on page: %s", body)
	}
	if strings.Contains(body, "<script") {
		t.Fatal("unexpected script injection path")
	}
}

type errSample string

func (e errSample) Error() string { return string(e) }
