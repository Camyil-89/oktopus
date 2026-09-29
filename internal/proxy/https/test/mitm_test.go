package https_test

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"oktopus/internal/pki"
	proxyhttps "oktopus/internal/proxy/https"
)

func testCA(t *testing.T) (*pki.Authority, *x509.CertPool) {
	dir := t.TempDir()
	if err := pki.GenerateCA(pki.CAConfig{
		CommonName: "oktopus test CA",
		ValidDays:  1,
		OutDir:     dir,
		Force:      true,
	}); err != nil {
		t.Fatal(err)
	}
	ca, err := pki.LoadAuthority(filepath.Join(dir, "ca.crt"), filepath.Join(dir, "ca.key"))
	if err != nil {
		t.Fatal(err)
	}
	pem, err := os.ReadFile(filepath.Join(dir, "ca.crt"))
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		t.Fatal("append CA")
	}
	return ca, pool
}

func TestMITMServe(t *testing.T) {
	if testing.Short() {
		t.Skip("MITM integration")
	}

	origin := httptest.NewTLSServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer origin.Close()

	originHost := origin.Listener.Addr().String()
	ca, pool := testCA(t)
	mitm := &proxyhttps.MITM{
		CA:                ca,
		OutboundTransport: origin.Client().Transport,
	}
	proxyAddr := startCONNECTProxy(t, mitm)

	conn := tls.Client(
		dialCONNECT(t, proxyAddr, originHost),
		&tls.Config{
			RootCAs:    pool,
			ServerName: "127.0.0.1",
			MinVersion: tls.VersionTLS12,
		},
	)
	defer conn.Close()
	if err := conn.Handshake(); err != nil {
		t.Fatal(err)
	}

	req, err := stdhttp.NewRequest(stdhttp.MethodGet, "http://"+originHost+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := req.Write(conn); err != nil {
		t.Fatal(err)
	}
	res, err := stdhttp.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != stdhttp.StatusNoContent {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status: %d body: %q", res.StatusCode, body)
	}
}

func BenchmarkMITMServe(b *testing.B) {
	if testing.Short() {
		b.Skip("MITM integration")
	}

	origin := httptest.NewTLSServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	b.Cleanup(origin.Close)

	originHost := origin.Listener.Addr().String()
	dir := b.TempDir()
	if err := pki.GenerateCA(pki.CAConfig{
		CommonName: "bench CA", ValidDays: 1, OutDir: dir, Force: true,
	}); err != nil {
		b.Fatal(err)
	}
	ca, err := pki.LoadAuthority(filepath.Join(dir, "ca.crt"), filepath.Join(dir, "ca.key"))
	if err != nil {
		b.Fatal(err)
	}
	pem, _ := os.ReadFile(filepath.Join(dir, "ca.crt"))
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(pem)

	mitm := &proxyhttps.MITM{
		CA:                ca,
		OutboundTransport: origin.Client().Transport,
	}
	proxyAddr := startCONNECTProxyBench(b, mitm)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		raw := dialCONNECTBench(b, proxyAddr, originHost)
		conn := tls.Client(raw, &tls.Config{
			RootCAs: pool, ServerName: "127.0.0.1", MinVersion: tls.VersionTLS12,
		})
		if err := conn.Handshake(); err != nil {
			b.Fatal(err)
		}
		req, _ := stdhttp.NewRequest(stdhttp.MethodGet, "http://"+originHost+"/", nil)
		_ = req.Write(conn)
		res, err := stdhttp.ReadResponse(bufio.NewReader(conn), req)
		if err != nil {
			conn.Close()
			b.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		res.Body.Close()
		conn.Close()
	}
}
