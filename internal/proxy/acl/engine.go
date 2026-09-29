package acl

import (
	"net"
	"regexp"

	"oktopus/internal/proxy/ratelimit"
)

type compiledPattern struct {
	sniFast sniFastKind
	sniHost string
	re      *regexp.Regexp
	ipFast  ipFastKind
	ipExact net.IP
	ipNet   *net.IPNet
	ipFrom  net.IP
	ipTo    net.IP
	portFast  portFastKind
	portExact int
	portFrom  int
	portTo    int
}

// Engine — скомпилированная Squid-подобная политика (http_access и связанные директивы).
type Engine struct {
	squidAccess      []SquidHTTPAccess
	squidSSLVerify   []SquidSSLVerify
	squidDelayAccess []SquidDelayAccess
	rateLimit        *ratelimit.Runtime
	squidFast        squidFastPath
}

// EmptyEngine — engine без правил (default deny в Decide).
func EmptyEngine() *Engine {
	return &Engine{}
}
