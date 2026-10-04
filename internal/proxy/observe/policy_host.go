package observe

import (
	"net"
	stdhttp "net/http"
	"strings"
)

// PolicyHostFromRequest — host origin для ACL, ssl_verify и inspect: URL.Host,
// затем Host, затем SNI из контекста MITM (CONNECT-туннель).
func PolicyHostFromRequest(req *stdhttp.Request, ctxSNI string) string {
	if req != nil && req.URL != nil {
		if h := strings.TrimSpace(req.URL.Host); h != "" {
			return hostOnly(h)
		}
	}
	if req != nil {
		if h := strings.TrimSpace(req.Host); h != "" {
			return hostOnly(h)
		}
	}
	return hostOnly(ctxSNI)
}

func hostOnly(hostPort string) string {
	hostPort = strings.TrimSpace(hostPort)
	if hostPort == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}
	return host
}

func normalizeTraceHost(hostPort string) string {
	return hostOnly(strings.TrimSpace(hostPort))
}
