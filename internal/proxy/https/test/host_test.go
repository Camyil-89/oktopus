package https_test

import (
	"testing"

	proxyhttps "oktopus/internal/proxy/https"
)

func TestNormalizeHostPort(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"example.com", "example.com:443"},
		{"example.com:8443", "example.com:8443"},
		{"127.0.0.1:8080", "127.0.0.1:8080"},
	}
	for _, tt := range tests {
		got := proxyhttps.NormalizeHostPort(tt.in)
		if got != tt.want {
			t.Fatalf("%q: got %q want %q", tt.in, got, tt.want)
		}
	}
}

func TestHostname(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"example.com:443", "example.com"},
		{"127.0.0.1:8080", "127.0.0.1"},
		{"nocolon", "nocolon"},
	}
	for _, tt := range tests {
		got := proxyhttps.Hostname(tt.in)
		if got != tt.want {
			t.Fatalf("%q: got %q want %q", tt.in, got, tt.want)
		}
	}
}

func BenchmarkNormalizeHostPort(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = proxyhttps.NormalizeHostPort("example.com")
	}
}

func BenchmarkHostname(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = proxyhttps.Hostname("example.com:443")
	}
}
