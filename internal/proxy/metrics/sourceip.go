package metrics

import (
	"net"
	"strings"
)

// SourceIP возвращает IP клиента без порта (для уникальных источников).
func SourceIP(hostPort string) string {
	hostPort = strings.TrimSpace(hostPort)
	if hostPort == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		host = hostPort
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return host
}
