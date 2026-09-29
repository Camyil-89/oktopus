package http_test

import (
	stdhttp "net/http"
	"testing"

	proxyhttp "oktopus/internal/proxy/http"
)

func TestStripHopByHopHeaders(t *testing.T) {
	tests := []struct {
		name string
		in   stdhttp.Header
		want map[string][]string
	}{
		{
			name: "removes standard hop-by-hop",
			in: stdhttp.Header{
				"Host":              {"example.com"},
				"Connection":        {"close"},
				"Transfer-Encoding": {"chunked"},
				"X-Custom":          {"keep"},
			},
			want: map[string][]string{
				"Host":     {"example.com"},
				"X-Custom": {"keep"},
			},
		},
		{
			name: "empty",
			in:   stdhttp.Header{},
			want: map[string][]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := cloneHeader(tt.in)
			proxyhttp.StripHopByHopHeaders(h)
			assertHeaderEqual(t, h, tt.want)
		})
	}
}

func TestCopyHeaders(t *testing.T) {
	src := stdhttp.Header{
		"Content-Type": {"text/plain"},
		"X-Multi":      {"a", "b"},
	}
	dst := make(stdhttp.Header)

	proxyhttp.CopyHeaders(dst, src)

	assertHeaderEqual(t, dst, map[string][]string{
		"Content-Type": {"text/plain"},
		"X-Multi":      {"a", "b"},
	})

	dst.Set("Content-Type", "old")
	proxyhttp.CopyHeaders(dst, src)
	if got := dst["Content-Type"]; len(got) != 2 {
		t.Fatalf("Content-Type values: %v", got)
	}
}

func BenchmarkStripHopByHopHeaders(b *testing.B) {
	h := stdhttp.Header{
		"Host":              {"example.com"},
		"Connection":        {"keep-alive, Foo"},
		"Keep-Alive":        {"timeout=5"},
		"Transfer-Encoding": {"chunked"},
		"Te":                {"trailers"},
		"X-Request-Id":      {"abc"},
	}
	b.ReportAllocs()
	for b.Loop() {
		proxyhttp.StripHopByHopHeaders(h)
		h.Set("Connection", "keep-alive, Foo")
		h.Set("Keep-Alive", "timeout=5")
		h.Set("Transfer-Encoding", "chunked")
		h.Set("Te", "trailers")
	}
}

func BenchmarkCopyHeaders(b *testing.B) {
	src := stdhttp.Header{
		"Content-Type":    {"application/json"},
		"Cache-Control":   {"no-cache"},
		"X-Forwarded-For": {"1.2.3.4", "5.6.7.8"},
	}
	b.ReportAllocs()
	for b.Loop() {
		dst := make(stdhttp.Header)
		proxyhttp.CopyHeaders(dst, src)
	}
}
