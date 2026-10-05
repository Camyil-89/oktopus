package https

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	stdhttp "net/http"
	"strings"

	"oktopus/internal/proxy/hooks"
	"oktopus/internal/proxy/observe"
	"oktopus/internal/proxy/outboundtls"
	proxyhttp "oktopus/internal/proxy/http"
	"oktopus/internal/proxy/wsproxy"
)

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
	reqCtx := observe.WithCONNECTDestHostPort(ctx, hostPort)
	reqCtx = observe.WithMITMClientHelloSNI(reqCtx, defaultHost)
	req = req.WithContext(reqCtx)

	outReq, d := applyMITMHTTPPolicy(ctx, m.Hooks, req, defaultHost)
	outReq.Header.Del("Proxy-Connection")

	if d.Handled() {
		_ = d.WriteResponseConn(clientConn, outReq)
		return true
	}

	sink := &mitmWSSink{conn: clientConn, br: br}
	continueSession, err := wsproxy.Proxy(ctx, wsproxy.Config{
		Hooks:     m.Hooks,
		RateLimit: m.RateLimit,
		Dial: func(ctx context.Context, outReq *stdhttp.Request) (net.Conn, error) {
			return m.dialUpstream(outReq.Context(), hostPort, defaultHost)
		},
	}, outReq, sink, wsproxy.RelayBlock)
	if err != nil {
		_ = WriteProxyError(outReq.Context(), clientConn, outReq, err, m.AccessLog)
		return true
	}
	return continueSession
}

type mitmWSSink struct {
	conn *tls.Conn
	br   *bufio.Reader
}

func (s *mitmWSSink) WritePolicy(ctx context.Context, outReq *stdhttp.Request, d hooks.Decision) error {
	return d.WriteResponseConn(s.conn, outReq)
}

func (s *mitmWSSink) DeliverUpstream(ctx context.Context, outReq *stdhttp.Request, res *stdhttp.Response) (wsproxy.RelayLegs, error) {
	if res.StatusCode != stdhttp.StatusSwitchingProtocols {
		proxyhttp.PrepareWebSocketOriginResponse(outReq, res)
		_ = res.Write(s.conn)
		return wsproxy.RelayLegs{}, nil
	}

	proxyhttp.PrepareWebSocketOriginResponse(outReq, res)
	if err := res.Write(s.conn); err != nil {
		return wsproxy.RelayLegs{}, err
	}

	return wsproxy.RelayLegs{
		ClientWrite: s.conn,
		ClientRead:  s.br,
		Switching:   true,
	}, nil
}

func prepareMITMOutboundRequest(ctx context.Context, req *stdhttp.Request, defaultHost string) *stdhttp.Request {
	base := ctx
	if req != nil && req.Context() != nil {
		base = req.Context()
	}
	outReq := req.WithContext(base)
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