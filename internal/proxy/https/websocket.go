package https

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"net"
	stdhttp "net/http"
	"strings"
	"sync"

	proxyhttp "oktopus/internal/proxy/http"
	"oktopus/internal/proxy/metrics"
	"oktopus/internal/proxy/observe"
	"oktopus/internal/proxy/outboundtls"
	"oktopus/internal/proxy/ratelimit"
)

// isWebSocketUpgrade — HTTP GET с Connection: Upgrade и Upgrade: websocket (RFC 6455).
func isWebSocketUpgrade(req *stdhttp.Request) bool {
	if req == nil || req.Method != stdhttp.MethodGet {
		return false
	}
	if !headerTokenListContains(req.Header, "Connection", "upgrade") {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(req.Header.Get("Upgrade")), "websocket")
}

func headerTokenListContains(h stdhttp.Header, key, want string) bool {
	for _, part := range strings.Split(h.Get(key), ",") {
		if strings.EqualFold(strings.TrimSpace(part), want) {
			return true
		}
	}
	return false
}

// serveWebSocketUpgrade проксирует handshake и затем сырые фреймы до закрытия соединения.
// Возвращает false, когда сессия MITM на этом TCP должна завершиться.
func (m *MITM) serveWebSocketUpgrade(
	ctx context.Context,
	clientConn *tls.Conn,
	br *bufio.Reader,
	req *stdhttp.Request,
	defaultHost string,
	hostPort string,
) bool {
	outReq, d := applyMITMHTTPPolicy(ctx, m.Hooks, req, defaultHost)
	outReq.Header.Del("Proxy-Connection")

	if d.Handled() {
		_ = d.WriteResponseConn(clientConn, outReq)
		return true
	}

	upstream, err := m.dialUpstream(outReq.Context(), hostPort, defaultHost)
	if err != nil {
		_ = WriteProxyError(outReq.Context(), clientConn, outReq, err, m.AccessLog)
		return true
	}

	if err := outReq.Write(upstream); err != nil {
		upstream.Close()
		_ = WriteProxyError(outReq.Context(), clientConn, outReq, err, m.AccessLog)
		return true
	}

	upBR := bufio.NewReader(upstream)
	outRes, err := stdhttp.ReadResponse(upBR, outReq)
	if err != nil {
		upstream.Close()
		_ = WriteProxyError(outReq.Context(), clientConn, outReq, err, m.AccessLog)
		return true
	}
	defer outRes.Body.Close()

	if d := m.Hooks.RunHTTPResponse(outReq.Context(), outReq, outRes); d.Handled() {
		upstream.Close()
		_ = d.WriteResponseConn(clientConn, outReq)
		return true
	}

	if outRes.StatusCode != stdhttp.StatusSwitchingProtocols {
		proxyhttp.StripHopByHopHeaders(outRes.Header)
		proxyhttp.StripHTTP3Hints(outRes.Header)
		outRes.ProtoMajor = 1
		outRes.ProtoMinor = 1
		outRes.Request = outReq
		_ = outRes.Write(clientConn)
		return true
	}

	outRes.ProtoMajor = 1
	outRes.ProtoMinor = 1
	outRes.Request = outReq
	if err := outRes.Write(clientConn); err != nil {
		upstream.Close()
		return false
	}

	metrics.IncActiveWebSocket()
	defer metrics.DecActiveWebSocket()
	var flow *ratelimit.Flow
	if m.RateLimit != nil {
		flow = m.RateLimit.DelayFlowHTTP(outReq.Context(), outReq)
	}
	relayPair(clientConn, br, upstream, upBR, flow)
	return false
}

func prepareMITMOutboundRequest(ctx context.Context, req *stdhttp.Request, defaultHost string) *stdhttp.Request {
	outReq := req.WithContext(ctx)
	outReq.URL.Scheme = "https"
	if outReq.URL.Host == "" {
		outReq.URL.Host = req.Host
	}
	if outReq.URL.Host == "" {
		outReq.URL.Host = defaultHost
	}
	outReq.RequestURI = ""
	policyHost := Hostname(outReq.URL.Host)
	if policyHost == "" {
		policyHost = defaultHost
	}
	outReq = outReq.WithContext(observe.WithSNI(outReq.Context(), policyHost))
	return outReq
}

func (m *MITM) dialUpstream(ctx context.Context, hostPort, serverName string) (net.Conn, error) {
	addr := hostPort
	if !strings.Contains(addr, ":") {
		addr += ":443"
	}
	if tr, ok := m.outboundTransport().(*stdhttp.Transport); ok && tr.DialTLSContext != nil {
		return tr.DialTLSContext(ctx, "tcp", addr)
	}
	var tlsCfg *tls.Config
	if tr, ok := m.outboundTransport().(*stdhttp.Transport); ok && tr.TLSClientConfig != nil {
		tlsCfg = tr.TLSClientConfig.Clone()
	}
	return dialTLSUpstream(ctx, addr, serverName, tlsCfg)
}

func dialTLSUpstream(ctx context.Context, hostPort, serverName string, tlsCfg *tls.Config) (net.Conn, error) {
	if serverName == "" {
		serverName, _, _ = net.SplitHostPort(hostPort)
	}
	tcp, err := proxyhttp.DialOutboundContext(ctx, "tcp", hostPort)
	if err != nil {
		return nil, err
	}
	cfg := mergeOutboundTLSConfig(ctx, serverName, tlsCfg)
	tlsConn := tls.Client(tcp, cfg)
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		tcp.Close()
		return nil, err
	}
	return tlsConn, nil
}

type relayEndpoint struct {
	r io.Reader
	c io.Closer
}

func (e *relayEndpoint) Read(p []byte) (int, error) {
	return e.r.Read(p)
}

func (e *relayEndpoint) Close() error {
	if e.c == nil {
		return nil
	}
	return e.c.Close()
}

func mergeOutboundTLSConfig(ctx context.Context, serverName string, tlsCfg *tls.Config) *tls.Config {
	var cfg *tls.Config
	if tlsCfg != nil {
		cfg = tlsCfg.Clone()
	} else {
		cfg = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	if cfg.ServerName == "" {
		cfg.ServerName = serverName
	}
	if cfg.MinVersion == 0 {
		cfg.MinVersion = tls.VersionTLS12
	}
	if verify, ok := outboundtls.VerifyFromContext(ctx); ok {
		cfg.InsecureSkipVerify = !verify
	}
	return cfg
}

func relayPair(clientConn *tls.Conn, clientBR *bufio.Reader, upstream net.Conn, upstreamBR *bufio.Reader, flow *ratelimit.Flow) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		Relay(upstream, &relayEndpoint{r: clientBR, c: clientConn}, flow, RelayFromClient, false)
		wg.Done()
	}()
	go func() {
		Relay(clientConn, &relayEndpoint{r: upstreamBR, c: upstream}, flow, RelayToClient, false)
		wg.Done()
	}()
	wg.Wait()
}
