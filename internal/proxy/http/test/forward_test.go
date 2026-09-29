package http_test

import (
	"context"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"oktopus/internal/proxy/hooks"
	proxyhttp "oktopus/internal/proxy/http"
)

func TestForwarderServe(t *testing.T) {
	const body = "origin-response-body"
	origin := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.Method != stdhttp.MethodGet {
			stdhttp.Error(w, "method", stdhttp.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/resource" {
			stdhttp.Error(w, "path", stdhttp.StatusNotFound)
			return
		}
		if r.Header.Get("Proxy-Connection") != "" {
			stdhttp.Error(w, "proxy header leaked", stdhttp.StatusBadRequest)
			return
		}
		w.Header().Set("Alt-Svc", `h3=":443"`)
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(body))
	}))
	defer origin.Close()

	f := &proxyhttp.Forwarder{Client: origin.Client()}

	inbound, err := stdhttp.NewRequest(stdhttp.MethodGet, origin.URL+"/resource", nil)
	if err != nil {
		t.Fatal(err)
	}
	inbound.Header.Set("Proxy-Connection", "keep-alive")
	inbound.Header.Set("Connection", "close")

	rec := httptest.NewRecorder()
	if err := f.Serve(context.Background(), rec, inbound); err != nil {
		t.Fatal(err)
	}
	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != stdhttp.StatusOK {
		t.Fatalf("status: %d", res.StatusCode)
	}
	if res.Header.Get("Alt-Svc") != "" {
		t.Fatalf("Alt-Svc should be stripped, got %q", res.Header.Get("Alt-Svc"))
	}
	gotBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBody) != body {
		t.Fatalf("body: got %q want %q", gotBody, body)
	}
}

func TestForwarderServe_relativeURL(t *testing.T) {
	f := &proxyhttp.Forwarder{Client: stdhttp.DefaultClient}
	req := httptest.NewRequest(stdhttp.MethodGet, "/relative", nil)
	err := f.Serve(context.Background(), httptest.NewRecorder(), req)
	if err == nil {
		t.Fatal("expected error for relative URL")
	}
	if !strings.Contains(err.Error(), "absolute URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestForwarderServe_hooks(t *testing.T) {
	origin := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusNoContent)
	}))
	defer origin.Close()

	var reqHits, resHits atomic.Int32
	h := &hooks.Hooks{
		OnHTTPRequest: func(_ context.Context, req *stdhttp.Request) hooks.Decision {
			reqHits.Add(1)
			if req.Header.Get("Proxy-Connection") != "" {
				t.Error("hook saw Proxy-Connection")
			}
			return hooks.AllowDecision()
		},
		OnHTTPResponse: func(_ context.Context, _ *stdhttp.Request, res *stdhttp.Response) hooks.Decision {
			resHits.Add(1)
			if res.StatusCode != stdhttp.StatusNoContent {
				t.Errorf("status in hook: %d", res.StatusCode)
			}
			return hooks.AllowDecision()
		},
	}
	f := &proxyhttp.Forwarder{Client: origin.Client(), Hooks: h}

	req, err := stdhttp.NewRequest(stdhttp.MethodGet, origin.URL+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Serve(context.Background(), httptest.NewRecorder(), req); err != nil {
		t.Fatal(err)
	}
	if reqHits.Load() != 1 || resHits.Load() != 1 {
		t.Fatalf("hooks: req=%d res=%d", reqHits.Load(), resHits.Load())
	}
}

func TestForwarderServe_originError(t *testing.T) {
	f := &proxyhttp.Forwarder{Client: &stdhttp.Client{Transport: roundTripperFunc(func(*stdhttp.Request) (*stdhttp.Response, error) {
		return nil, io.EOF
	})}}
	req := httptest.NewRequest(stdhttp.MethodGet, "http://127.0.0.1:1/nope", nil)
	err := f.Serve(context.Background(), httptest.NewRecorder(), req)
	if err == nil {
		t.Fatal("expected transport error")
	}
}

type roundTripperFunc func(*stdhttp.Request) (*stdhttp.Response, error)

func (f roundTripperFunc) RoundTrip(r *stdhttp.Request) (*stdhttp.Response, error) {
	return f(r)
}

func BenchmarkForwarderServe(b *testing.B) {
	const payload = "ok"
	origin := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer origin.Close()

	f := &proxyhttp.Forwarder{Client: origin.Client()}
	req, err := stdhttp.NewRequest(stdhttp.MethodGet, origin.URL+"/bench", nil)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		rec := httptest.NewRecorder()
		if err := f.Serve(context.Background(), rec, req); err != nil {
			b.Fatal(err)
		}
	}
}
