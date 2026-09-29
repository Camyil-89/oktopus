package acl

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type portFastKind int

const (
	portFastNone portFastKind = iota
	portFastExact
	portFastRange
)

func portValid(port int) bool {
	return port >= 1 && port <= 65535
}

func portPatternMatches(p compiledPattern, port int) bool {
	if !portValid(port) {
		return false
	}
	switch p.portFast {
	case portFastExact:
		return port == p.portExact
	case portFastRange:
		return port >= p.portFrom && port <= p.portTo
	default:
		return false
	}
}

func parsePortPatternLine(pat string) (compiledPattern, error) {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return compiledPattern{}, fmt.Errorf("empty")
	}
	if i := strings.Index(pat, "-"); i > 0 {
		left := strings.TrimSpace(pat[:i])
		right := strings.TrimSpace(pat[i+1:])
		from, err1 := parsePortNumber(left)
		to, err2 := parsePortNumber(right)
		if err1 == nil && err2 == nil {
			if from > to {
				return compiledPattern{}, fmt.Errorf("range: start after end")
			}
			return compiledPattern{
				portFast: portFastRange,
				portFrom: from,
				portTo:   to,
			}, nil
		}
	}
	port, err := parsePortNumber(pat)
	if err != nil {
		return compiledPattern{}, fmt.Errorf("invalid port or range")
	}
	return compiledPattern{
		portFast:  portFastExact,
		portExact: port,
	}, nil
}

func parsePortNumber(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	n, err := strconv.Atoi(s)
	if err != nil || !portValid(n) {
		return 0, fmt.Errorf("invalid port")
	}
	return n, nil
}

// DstPortForMatch — порт назначения: явное поле или из host:port в SNI/Host.
func DstPortForMatch(f RequestFields) int {
	if portValid(f.DstPort) {
		return f.DstPort
	}
	host := strings.TrimSpace(f.SNI)
	if host == "" {
		return 0
	}
	if _, ps, err := net.SplitHostPort(host); err == nil {
		if p, err := parsePortNumber(ps); err == nil {
			return p
		}
	}
	return 0
}

// ParseDstPort извлекает порт из строки host:port (CONNECT, Host с портом).
func ParseDstPort(hostPort string) int {
	hostPort = strings.TrimSpace(hostPort)
	if hostPort == "" {
		return 0
	}
	_, ps, err := net.SplitHostPort(hostPort)
	if err != nil {
		return 0
	}
	p, err := parsePortNumber(ps)
	if err != nil {
		return 0
	}
	return p
}
