package https

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"strings"
	"time"

	"sync"

	"oktopus/internal/pki"
	"oktopus/internal/proxy/accesslog"
	"oktopus/internal/proxy/gateway"
	proxyhttp "oktopus/internal/proxy/http"
	"oktopus/internal/proxy/bytecount"
	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/wsproxy"
)

// MITM — CONNECT с расшифровкой TLS (нужен установленный ca.crt на клиенте).
type MITM struct {
	CA         *pki.Authority
	Hooks      *hooks.Hooks
	AccessLog  accesslog.Recorder
	RateLimit  hooks.RateLimitPolicy
	// OutboundTransport — исходящие запросы к origin; если nil — один общий Transport на экземпляр MITM.
	OutboundTransport stdhttp.RoundTripper

	outboundOnce sync.Once
	outbound     stdhttp.RoundTripper
}

// Serve отвечает на CONNECT и обрабатывает сессию в отдельной горутине.
func (m *MITM) Serve(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request) error {
	hostPort, proceed, err := serveConnectAfterACL(ctx, w, r, m.Hooks, m.CA)
	if err != nil || !proceed {
		return err
	}
	return m.serveConnectEstablished(ctx, w, r, hostPort, false)
}

func (m *MITM) serveConnectEstablished(ctx context.Context, w stdhttp.ResponseWriter, r *stdhttp.Request, hostPort string, sessionDenied bool) error {
	host := Hostname(hostPort)

	hj, ok := w.(stdhttp.Hijacker)
	if !ok {
		return fmt.Errorf("hijack not supported")
	}

	rawClient, bufrw, err := hj.Hijack()
	if err != nil {
		return err
	}

	if _, err = bufrw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		rawClient.Close()
		return err
	}
	if err := bufrw.Flush(); err != nil {
		rawClient.Close()
		return err
	}

	go m.runSession(ctx, rawClient, bufrw, host, hostPort, sessionDenied)
	return nil
}

func (m *MITM) runSession(ctx context.Context, rawClient net.Conn, bufrw *bufio.ReadWriter, fallbackHost, hostPort string, sessionDenied bool) {
	defer rawClient.Close()

	conn := bytecount.WrapConnPolicy(newHijackedConn(rawClient, bufrw), sessionDenied)

	tlsClient := tls.Server(conn, &tls.Config{
		MinVersion:             tls.VersionTLS12,
		NextProtos:             []string{"http/1.1"},
		SessionTicketsDisabled: true, // tickets + динамический GetCertificate ломают resume/таймауты
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			name := chi.ServerName
			if name == "" {
				name = fallbackHost
			}
			return m.CA.LeafCertificate(name)
		},
	})

	handshakeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tlsClient.HandshakeContext(handshakeCtx); err != nil {
		return
	}

	sni := tlsClient.ConnectionState().ServerName
	if sni == "" {
		sni = fallbackHost
	}

	client := &stdhttp.Client{
		Transport: m.outboundTransport(),
		CheckRedirect: func(req *stdhttp.Request, via []*stdhttp.Request) error {
			return stdhttp.ErrUseLastResponse
		},
	}

	br := bufio.NewReader(tlsClient)
	for {
		if !m.serveOneRequest(ctx, tlsClient, br, sni, client, hostPort) {
			return
		}
	}
}

func (m *MITM) serveOneRequest(ctx context.Context, clientConn *tls.Conn, br *bufio.Reader, defaultHost string, client *stdhttp.Client, hostPort string) bool {
	req, err := stdhttp.ReadRequest(br)
	if err != nil {
		return false
	}
	defer req.Body.Close()

	if wsproxy.IsWebSocketUpgrade(req) {
		return m.serveWebSocketUpgrade(ctx, clientConn, br, req, defaultHost, hostPort)
	}

	outReq, d := applyMITMHTTPPolicy(ctx, m.Hooks, req, defaultHost)

	proxyhttp.StripHopByHopHeaders(outReq.Header)

	if d.Handled() {
		_ = d.WriteResponseConn(clientConn, outReq)
		return true
	}

	if m.RateLimit != nil {
		if flow := m.RateLimit.DelayFlowHTTP(outReq.Context(), outReq); flow != nil && outReq.Body != nil && outReq.Body != stdhttp.NoBody {
			outReq.Body = rateLimitReadCloser{Reader: flow.Reader(outReq.Body), closer: outReq.Body}
		}
	}

	outRes, err := client.Do(outReq)
	if err != nil {
		_ = WriteProxyError(outReq.Context(), clientConn, outReq, err, m.AccessLog)
		return true
	}
	defer outRes.Body.Close()

	if d := m.Hooks.RunHTTPResponse(outReq.Context(), outReq, outRes); d.Handled() {
		_ = d.WriteResponseConn(clientConn, outReq)
		return true
	}

	proxyhttp.StripHopByHopHeaders(outRes.Header)
	proxyhttp.StripHTTP3Hints(outRes.Header)
	if m.RateLimit != nil {
		if flow := m.RateLimit.DelayFlowHTTP(outReq.Context(), outReq); flow != nil && outRes.Body != nil {
			outRes.Body = rateLimitReadCloser{Reader: flow.Reader(outRes.Body), closer: outRes.Body}
		}
	}
	outRes.ProtoMajor = 1
	outRes.ProtoMinor = 1
	outRes.Request = outReq
	if err := outRes.Write(clientConn); err != nil {
		return false
	}
	return true
}

// WriteProxyError отвечает клиенту MITM HTTP/1.1 502 (HTML-страница шлюза).
func WriteProxyError(ctx context.Context, w io.Writer, req *stdhttp.Request, err error, rec accesslog.Recorder) error {
	page := gateway.RecordGatewayError(ctx, rec, req, err)
	res := &stdhttp.Response{
		StatusCode:    stdhttp.StatusBadGateway,
		Status:        "502 Bad Gateway",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        stdhttp.Header{"Content-Type": []string{gateway.ContentTypeHTML()}},
		Body:          io.NopCloser(strings.NewReader(string(page.Body))),
		ContentLength: int64(len(page.Body)),
	}
	return res.Write(w)
}

func (m *MITM) outboundTransport() stdhttp.RoundTripper {
	if m.OutboundTransport != nil {
		return m.OutboundTransport
	}
	m.outboundOnce.Do(func() {
		m.outbound = proxyhttp.NewTLSOutboundTransport()
	})
	return m.outbound
}

type rateLimitReadCloser struct {
	io.Reader
	closer io.Closer
}

func (r rateLimitReadCloser) Close() error {
	if r.closer == nil {
		return nil
	}
	return r.closer.Close()
}
