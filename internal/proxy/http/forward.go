package http

import (
	"context"
	"fmt"
	stdhttp "net/http"

	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/bytecount"
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

	outReq := inbound.Clone(ctx)
	outReq.RequestURI = ""
	StripHopByHopHeaders(outReq.Header)
	outReq.Header.Del("Proxy-Connection")

	if d := f.Hooks.RunHTTPRequest(ctx, outReq); d.Handled() {
		denied := !d.Allow
		w = bytecount.WrapResponseWriterPolicy(w, denied)
		if denied {
			bytecount.ObserveRequestLineHeadersDenied(inbound)
		} else {
			bytecount.ObserveRequestLineHeaders(inbound)
		}
		d.WriteResponse(w, inbound)
		return nil
	}

	w = bytecount.WrapResponseWriter(w)
	bytecount.ObserveRequestLineHeaders(inbound)

	if outReq.Body != nil && outReq.Body != stdhttp.NoBody {
		outReq.Body = bytecount.WrapBodyUp(outReq.Body)
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
