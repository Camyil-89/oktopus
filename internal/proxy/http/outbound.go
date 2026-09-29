package http

import (
	"context"
	"crypto/tls"
	"net"
	stdhttp "net/http"
	"strings"
	"time"

	"oktopus/internal/proxy/outboundtls"
)

const (
	defaultMaxIdleConns        = 32768
	defaultMaxIdleConnsPerHost = 256
	defaultMaxConnsPerHost     = 2048
	defaultIdleConnTimeout     = 90 * time.Second
)

// DialOutboundContext — TCP к origin. При network=tcp используется tcp4, чтобы не ломаться
// в Docker/сетях без маршрута IPv6 (AAAA → network is unreachable).
func DialOutboundContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if network == "tcp" || network == "tcp6" {
		network = "tcp4"
	}
	d := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return d.DialContext(ctx, network, addr)
}

// NewOutboundTransport — общий исходящий transport для cleartext http:// к origin.
func NewOutboundTransport() *stdhttp.Transport {
	return &stdhttp.Transport{
		Proxy:                 nil,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          defaultMaxIdleConns,
		MaxIdleConnsPerHost:   defaultMaxIdleConnsPerHost,
		MaxConnsPerHost:       defaultMaxConnsPerHost,
		IdleConnTimeout:       defaultIdleConnTimeout,
		DisableCompression:    true,
		ResponseHeaderTimeout: 0,
		DialContext:           DialOutboundContext,
	}
}

// NewTLSOutboundTransport — исходящий transport для MITM (HTTP/1.1 поверх TLS к origin).
func NewTLSOutboundTransport() *stdhttp.Transport {
	t := NewOutboundTransport()
	t.TLSClientConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	t.DialTLSContext = dialTLSContext
	return t
}

func dialTLSContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if !strings.Contains(addr, ":") {
		addr += ":443"
	}
	host, _, _ := net.SplitHostPort(addr)
	tcp, err := DialOutboundContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	tlsConn := tls.Client(tcp, outboundtls.ClientConfig(ctx, host))
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		tcp.Close()
		return nil, err
	}
	return tlsConn, nil
}

// TLSClientConfigForContext — alias для outboundtls.ClientConfig (websocket и тесты).
func TLSClientConfigForContext(ctx context.Context, serverName string) *tls.Config {
	return outboundtls.ClientConfig(ctx, serverName)
}
