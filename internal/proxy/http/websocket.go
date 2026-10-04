package http

import (
	"bufio"
	"context"
	"fmt"
	"net"
	stdhttp "net/http"
	"strings"

	"oktopus/internal/proxy/bytecount"
	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/wsproxy"
)

func (f *Forwarder) serveWebSocketUpgrade(ctx context.Context, w stdhttp.ResponseWriter, inbound *stdhttp.Request) error {
	outReq := inbound.Clone(ctx)
	outReq.RequestURI = ""
	outReq.Header.Del("Proxy-Connection")

	instID := instanceIDFromCtx(ctx)
	if d := f.Hooks.RunHTTPRequest(ctx, outReq); d.Handled() {
		denied := !d.Allow
		w = bytecount.WrapResponseWriterPolicyInstance(w, denied, instID)
		bytecount.ObserveRequestLineHeadersInstance(inbound, instID, denied)
		d.WriteResponse(w, inbound)
		return nil
	}
	bytecount.ObserveRequestLineHeadersInstance(inbound, instID, false)

	sink := &hijackSink{w: w, inbound: inbound}
	_, err := wsproxy.Proxy(ctx, wsproxy.Config{
		Hooks:     f.Hooks,
		RateLimit: f.RateLimit,
		Dial: func(ctx context.Context, outReq *stdhttp.Request) (net.Conn, error) {
			return DialOutboundContext(ctx, "tcp", originTCPAddr(outReq))
		},
	}, outReq, sink, wsproxy.RelayAsync)
	return err
}

type hijackSink struct {
	w       stdhttp.ResponseWriter
	inbound *stdhttp.Request
}

func (s *hijackSink) WritePolicy(ctx context.Context, outReq *stdhttp.Request, d hooks.Decision) error {
	d.WriteResponse(s.w, s.inbound)
	return nil
}

func (s *hijackSink) DeliverUpstream(ctx context.Context, outReq *stdhttp.Request, res *stdhttp.Response) (wsproxy.RelayLegs, error) {
	hj, ok := s.w.(stdhttp.Hijacker)
	if !ok {
		return wsproxy.RelayLegs{}, fmt.Errorf("websocket upgrade: ResponseWriter is not Hijacker")
	}
	clientConn, bufrw, err := hj.Hijack()
	if err != nil {
		return wsproxy.RelayLegs{}, err
	}
	clientRead := newHijackedConn(clientConn, bufrw)

	if res.StatusCode != stdhttp.StatusSwitchingProtocols {
		PrepareWebSocketOriginResponse(outReq, res)
		_ = res.Write(clientConn)
		clientConn.Close()
		return wsproxy.RelayLegs{}, nil
	}

	PrepareWebSocketOriginResponse(outReq, res)
	if err := res.Write(clientConn); err != nil {
		clientConn.Close()
		return wsproxy.RelayLegs{}, err
	}

	return wsproxy.RelayLegs{
		ClientWrite: clientConn,
		ClientRead:  clientRead,
		Switching:   true,
	}, nil
}

// PrepareWebSocketOriginResponse нормализует ответ origin перед записью клиенту (101 или ошибка).
func PrepareWebSocketOriginResponse(outReq *stdhttp.Request, res *stdhttp.Response) {
	if res.StatusCode != stdhttp.StatusSwitchingProtocols {
		StripHopByHopHeaders(res.Header)
		StripHTTP3Hints(res.Header)
	}
	res.ProtoMajor = 1
	res.ProtoMinor = 1
	res.Request = outReq
}

func originTCPAddr(req *stdhttp.Request) string {
	host := req.URL.Host
	if host == "" {
		host = req.Host
	}
	if _, _, err := net.SplitHostPort(host); err != nil {
		if strings.EqualFold(req.URL.Scheme, "https") {
			host = net.JoinHostPort(host, "443")
		} else {
			host = net.JoinHostPort(host, "80")
		}
	}
	return host
}

type hijackedConn struct {
	net.Conn
	reader *bufio.Reader
}

func newHijackedConn(c net.Conn, rw *bufio.ReadWriter) net.Conn {
	return &hijackedConn{Conn: c, reader: rw.Reader}
}

func (c *hijackedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}
