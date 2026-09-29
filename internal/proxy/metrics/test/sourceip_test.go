package metrics_test

import (
	"testing"

	"oktopus/internal/proxy/metrics"
)

func TestSourceIP(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"192.168.1.10:54321", "192.168.1.10"},
		{"192.168.1.10:11111", "192.168.1.10"},
		{"[::1]:8080", "::1"},
		{"", ""},
	}
	for _, c := range cases {
		if got := metrics.SourceIP(c.in); got != c.want {
			t.Fatalf("SourceIP(%q) = %q want %q", c.in, got, c.want)
		}
	}
}
