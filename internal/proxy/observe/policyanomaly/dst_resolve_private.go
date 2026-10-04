package policyanomaly

import (
	"net"
	"strings"
)

const KindDstResolvePrivate = "dst_resolve_private"

func init() {
	Register(KindDstResolvePrivate, detectDstResolvePrivate)
}

func detectDstResolvePrivate(in Input) *CheckResult {
	host := normalizeHost(in.PolicyHost)
	if host == "" {
		host = normalizeHost(in.ConnectHostPort)
	}
	if host == "" || net.ParseIP(host) != nil {
		return nil
	}
	var sensitive []string
	for _, s := range in.DstResolved {
		s = strings.TrimSpace(s)
		ip := net.ParseIP(s)
		if ip == nil || !ipSensitiveForPolicy(ip) {
			continue
		}
		sensitive = append(sensitive, ip.String())
	}
	if len(sensitive) == 0 {
		return nil
	}
	return &CheckResult{
		Detect:      true,
		PolicyHost:  host,
		ResolvedIPs: sensitive,
	}
}
