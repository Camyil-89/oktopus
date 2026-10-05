package http

import (
	"context"
	"fmt"
	stdhttp "net/http"

	"oktopus/internal/proxy/bytecount"
	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/instancectx"

	"github.com/google/uuid"
	"oktopus/internal/proxy/wsproxy"
)

// Forwarder проксирует запросы с абсолютным URL (http://…).
type Forwarder struct {
	Client    *stdhttp.Client
	Hooks     *hooks.Hooks
	RateLimit hooks.RateLimitPolicy
}

// Serve обрабатывает один входящий proxy-запрос (не CONNECT).
func (f *Forwarder) Serve(ctx context.Context, w stdhttp.ResponseWriter, inbound *stdhttp.Request) error {
	if inbound.URL == nil || !inbound.URL.IsAbs() {
		return fmt.Errorf("proxy expects absolute URL, got %q", inbound.URL)
	}

	if wsproxy.IsWebSocketUpgrade(inbound) {
		return f.serveWebSocketUpgrade(ctx, w, inbound)
	}

	outReq := inbound.Clone(ctx)
	outReq.RequestURI = ""
	StripHopByHopHeaders(outReq.Header)
	outReq.Header.Del("Proxy-Connection")

	instID := instanceIDFromCtx(ctx)
	if d := f.Hooks.RunHTTPRequest(ctx, outReq); d.Handled() {
		denied := !d.Allow
		w = bytecount.WrapResponseWriterPolicyInstance(w, denied, instID)
		bytecount.ObserveRequestLineHeadersInstance(inbound, instID, denied)
		d.WriteResponse(w, inbound)
		return nil
	}

	w = bytecount.WrapResponseWriterPolicyInstance(w, false, instID)
	bytecount.ObserveRequestLineHeadersInstance(inbound, instID, false)

	if outReq.Body != nil && outReq.Body != stdhttp.NoBody {
		outReq.Body = bytecount.WrapBodyUpPolicyInstance(outReq.Body, false, instID)
	}
	if f.RateLimit != nil {
		if flow := f.RateLimit.DelayFlowHTTP(ctx, outReq); flow != nil && outReq.Body != nil && outReq.Body != stdhttp.NoBody {
			outReq.Body = rateLimitBody{Reader: flow.Reader(outReq.Body), closer: outReq.Body}
		}
	}

	outRes, err := f.Client.Do(outReq)
	if err != nil {
		return err
	}
	defer outRes.Body.Close()

	if d := f.Hooks.RunHTTPResponse(ctx, outReq, outRes); d.WriteResponse(w, inbound) {
		return nil
	}

	StripHopByHopHeaders(outRes.Header)
	StripHTTP3Hints(outRes.Header)
	CopyHeaders(w.Header(), outRes.Header)
	w.WriteHeader(outRes.StatusCode)
	body := outRes.Body
	if f.RateLimit != nil {
		if flow := f.RateLimit.DelayFlowHTTP(ctx, outReq); flow != nil && body != nil {
			body = rateLimitBody{Reader: flow.Reader(body), closer: body}
		}
	}
	_, err = Copy(w, body)
	return err
}

func instanceIDFromCtx(ctx context.Context) uuid.UUID {
	if id, ok := instancectx.ID(ctx); ok {
		return id
	}
	return uuid.Nil
}
