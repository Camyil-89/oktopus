package https

import (
	"net"
	"strings"
)

// NormalizeHostPort дополняет Host из CONNECT портом :443 при необходимости.
func NormalizeHostPort(host string) string {
	if host == "" {
		return ""
	}
	if !strings.Contains(host, ":") {
		return host + ":443"
	}
	return host
}

// Hostname извлекает имя хоста из host:port.
func Hostname(hostPort string) string {
	h, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}
	return h
}
