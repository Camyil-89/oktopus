package https_test

import (
	"bytes"
	"io"
	"testing"

	proxyhttps "oktopus/internal/proxy/https"
)

type nopCloser struct{ io.Reader }

func (nopCloser) Close() error { return nil }

type writeNopCloser struct {
	*bytes.Buffer
}

func (writeNopCloser) Close() error { return nil }

func TestRelay(t *testing.T) {
	payload := []byte("relay-payload-bytes")
	src := nopCloser{bytes.NewReader(payload)}
	dst := &writeNopCloser{bytes.NewBuffer(nil)}

	proxyhttps.Relay(dst, src, nil, proxyhttps.RelayToClient, true)

	if !bytes.Equal(dst.Buffer.Bytes(), payload) {
		t.Fatalf("got %q", dst.Buffer.Bytes())
	}
}

func BenchmarkRelay(b *testing.B) {
	const size = 32 << 10
	payload := bytes.Repeat([]byte("x"), size)
	b.SetBytes(int64(size))
	b.ReportAllocs()
	for b.Loop() {
		src := nopCloser{bytes.NewReader(payload)}
		dst := &writeNopCloser{bytes.NewBuffer(nil)}
		proxyhttps.Relay(dst, src, nil, proxyhttps.RelayToClient, true)
	}
}
