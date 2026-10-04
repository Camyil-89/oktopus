package policyanomaly

import (
	"net"
	"strconv"
	"strings"
)

func normalizeHost(hostPort string) string {
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

func hostsDiffer(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return !strings.EqualFold(a, b)
}

func tracePort(hostPort string) int {
	hostPort = strings.TrimSpace(hostPort)
	if hostPort == "" {
		return 0
	}
	_, portStr, err := net.SplitHostPort(hostPort)
	if err != nil {
		return 0
	}
	p, err := strconv.Atoi(portStr)
	if err != nil || p <= 0 || p > 65535 {
		return 0
	}
	return p
}

func httpPortFromInput(in Input) int {
	if in.HTTPPort != 0 {
		return in.HTTPPort
	}
	if p := tracePort(in.URLHost); p != 0 {
		return p
	}
	return tracePort(in.HTTPHostHeader)
}

func connectPortFromInput(in Input) int {
	if in.ConnectPort != 0 {
		return in.ConnectPort
	}
	return tracePort(in.ConnectHostPort)
}

func ipSensitiveForPolicy(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}
