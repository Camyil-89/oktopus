package http_test

import (
	stdhttp "net/http"
	"testing"

	proxyhttp "oktopus/internal/proxy/http"
)

func TestStripHTTP3Hints(t *testing.T) {
	h := make(stdhttp.Header)
	h.Set("Alt-Svc", `h3=":443"; ma=86400`)
	h.Set("Alt-Svc-Clear", "1")
	h.Set("HTTP2-Settings", "aabbcc")
	h.Set("Content-Type", "text/html")
	proxyhttp.StripHTTP3Hints(h)
	assertHeaderEqual(t, h, map[string][]string{
		"Content-Type": {"text/html"},
	})
}

func BenchmarkStripHTTP3Hints(b *testing.B) {
	h := make(stdhttp.Header)
	h.Set("Content-Type", "text/html")
	h.Set("Cache-Control", "max-age=0")
	b.ReportAllocs()
	for b.Loop() {
		h.Set("Alt-Svc", `h3=":443"; ma=86400`)
		h.Set("Alt-Svc-Clear", "1")
		h.Set("HTTP2-Settings", "aabbcc")
		proxyhttp.StripHTTP3Hints(h)
	}
}
