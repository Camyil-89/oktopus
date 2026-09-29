package http_test

import (
	stdhttp "net/http"
	"testing"
)

func cloneHeader(h stdhttp.Header) stdhttp.Header {
	out := make(stdhttp.Header, len(h))
	for k, vv := range h {
		cp := make([]string, len(vv))
		copy(cp, vv)
		out[k] = cp
	}
	return out
}

func assertHeaderEqual(t *testing.T, got stdhttp.Header, want map[string][]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("header count: got %d want %d\ngot=%v", len(got), len(want), got)
	}
	for k, wv := range want {
		gv, ok := got[k]
		if !ok {
			t.Fatalf("missing key %q", k)
		}
		if len(gv) != len(wv) {
			t.Fatalf("%q: got %v want %v", k, gv, wv)
		}
		for i := range wv {
			if gv[i] != wv[i] {
				t.Fatalf("%q[%d]: got %q want %q", k, i, gv[i], wv[i])
			}
		}
	}
}
