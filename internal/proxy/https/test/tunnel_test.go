package https_test

import (
	"bufio"
	"context"
	"io"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"oktopus/internal/proxy/hooks"
	proxyhttps "oktopus/internal/proxy/https"
)

func startTCPEchoServer() (addr string, stop func()) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	done := make(chan struct{})
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
				}
				continue
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}(c)
		}
	}()
	return ln.Addr().String(), func() {
		close(done)
		ln.Close()
	}
}

func startCONNECTProxy(t *testing.T, h proxyhttps.ConnectHandler) string {
	t.Helper()
	srv := httptest.NewUnstartedServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Method != stdhttp.MethodConnect {
			stdhttp.Error(w, "CONNECT only", stdhttp.StatusMethodNotAllowed)
			return
		}
		if err := h.Serve(context.WithoutCancel(r.Context()), w, r); err != nil {
			t.Errorf("connect: %v", err)
		}
	}))
	srv.Start()
	t.Cleanup(srv.Close)
	return srv.Listener.Addr().String()
}

func TestTunnelServe(t *testing.T) {
	backend, stopBackend := startTCPEchoServer()
	defer stopBackend()

	var connects atomic.Int32
	tunnel := &proxyhttps.Tunnel{
		Hooks: &hooks.Hooks{
			OnConnect: func(_ context.Context, hostPort string) hooks.Decision {
				connects.Add(1)
				if hostPort != backend {
					t.Errorf("OnConnect host: %q", hostPort)
				}
				return hooks.AllowDecision()
			},
		},
	}
	proxyAddr := startCONNECTProxy(t, tunnel)

	conn := dialCONNECT(t, proxyAddr, backend)
	defer conn.Close()

	const msg = "tunnel-hello"
	if _, err := io.WriteString(conn, msg); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != msg {
		t.Fatalf("echo: %q", buf)
	}
	if connects.Load() != 2 {
		t.Fatalf("OnConnect: %d (expected 2: ACL + PORT recheck)", connects.Load())
	}
}

func TestTunnelServe_noHost(t *testing.T) {
	tunnel := &proxyhttps.Tunnel{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodConnect, "https://example.com/", nil)
	req.Host = ""
	err := tunnel.Serve(context.Background(), rec, req)
	if err == nil {
		t.Fatal("expected error")
	}
}

func BenchmarkTunnelServe(b *testing.B) {
	if testing.Short() {
		b.Skip("CONNECT integration")
	}
	backend, stopBackend := startTCPEchoServer()
	b.Cleanup(stopBackend)

	tunnel := &proxyhttps.Tunnel{}
	proxyAddr := startCONNECTProxyBench(b, tunnel)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		conn := dialCONNECTBench(b, proxyAddr, backend)
		_, _ = io.WriteString(conn, "x")
		_, _ = io.ReadAll(conn)
		conn.Close()
	}
}

func startCONNECTProxyBench(b *testing.B, h proxyhttps.ConnectHandler) string {
	srv := httptest.NewUnstartedServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		_ = h.Serve(context.WithoutCancel(r.Context()), w, r)
	}))
	srv.Start()
	b.Cleanup(srv.Close)
	return srv.Listener.Addr().String()
}

func dialCONNECTBench(b *testing.B, proxyAddr, dest string) net.Conn {
	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		b.Fatal(err)
	}
	req := "CONNECT " + dest + " HTTP/1.1\r\nHost: " + dest + "\r\n\r\n"
	if _, err := io.WriteString(conn, req); err != nil {
		conn.Close()
		b.Fatal(err)
	}
	br := bufio.NewReader(conn)
	resp, err := stdhttp.ReadResponse(br, &stdhttp.Request{Method: stdhttp.MethodConnect})
	if err != nil {
		conn.Close()
		b.Fatal(err)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		conn.Close()
		b.Fatalf("status %d", resp.StatusCode)
	}
	return &bufConn{Conn: conn, r: br}
}
