package https_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"oktopus/internal/proxy/gateway"
	proxyhttps "oktopus/internal/proxy/https"
)

func TestWriteProxyError(t *testing.T) {
	_ = gateway.Reload()
	var buf bytes.Buffer
	err := proxyhttps.WriteProxyError(context.Background(), &buf, nil, ioEOF("upstream failed"), nil)
	if err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if !strings.Contains(s, "502 Bad Gateway") {
		t.Fatalf("response: %q", s)
	}
	if strings.Contains(s, "upstream failed") {
		t.Fatalf("upstream text must not be on page: %q", s)
	}
	if !strings.Contains(s, "text/html") {
		t.Fatalf("content-type: %q", s)
	}
}

type ioEOF string

func (e ioEOF) Error() string { return string(e) }

func BenchmarkWriteProxyError(b *testing.B) {
	err := ioEOF("benchmark error")
	b.ReportAllocs()
	for b.Loop() {
		var buf bytes.Buffer
		if e := proxyhttps.WriteProxyError(context.Background(), &buf, nil, err, nil); e != nil {
			b.Fatal(e)
		}
	}
}
