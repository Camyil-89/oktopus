package acl

import (
	"fmt"
	"net"
	"strings"
)

type ipFastKind int

const (
	ipFastNone ipFastKind = iota
	ipFastExact
	ipFastCIDR
	ipFastRange
)

func ipValid(ip net.IP) bool {
	return ip != nil && ip.To16() != nil
}

func ipKey(ip net.IP) string {
	ip = ip.To16()
	if ip == nil {
		return ""
	}
	return ip.String()
}

func ipTo16(ip net.IP) net.IP {
	if ip4 := ip.To4(); ip4 != nil {
		return ip4
	}
	return ip.To16()
}

func ipInRange(ip, from, to net.IP) bool {
	ip = ipTo16(ip)
	from = ipTo16(from)
	to = ipTo16(to)
	if ip == nil || from == nil || to == nil {
		return false
	}
	return compareIP(ip, from) >= 0 && compareIP(ip, to) <= 0
}

func compareIP(a, b net.IP) int {
	a = ipTo16(a)
	b = ipTo16(b)
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func ipPatternMatches(p compiledPattern, ip net.IP) bool {
	if !ipValid(ip) {
		return false
	}
	switch p.ipFast {
	case ipFastExact:
		return ipKey(ip) == ipKey(p.ipExact)
	case ipFastCIDR:
		return p.ipNet != nil && p.ipNet.Contains(ip)
	case ipFastRange:
		return ipInRange(ip, p.ipFrom, p.ipTo)
	default:
		return false
	}
}

func parseIPPatternLine(pat string) (compiledPattern, error) {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return compiledPattern{}, fmt.Errorf("empty")
	}
	if strings.Contains(pat, "/") {
		_, n, err := net.ParseCIDR(pat)
		if err != nil {
			return compiledPattern{}, fmt.Errorf("cidr: %w", err)
		}
		return compiledPattern{
			ipFast: ipFastCIDR,
			ipNet:  n,
		}, nil
	}
	if i := strings.Index(pat, "-"); i > 0 {
		left := strings.TrimSpace(pat[:i])
		right := strings.TrimSpace(pat[i+1:])
		from := net.ParseIP(left)
		to := net.ParseIP(right)
		if from != nil && to != nil && sameIPFamily(from, to) {
			if compareIP(from, to) > 0 {
				return compiledPattern{}, fmt.Errorf("range: start after end")
			}
			return compiledPattern{
				ipFast: ipFastRange,
				ipFrom: from,
				ipTo:   to,
			}, nil
		}
	}
	ip := net.ParseIP(pat)
	if ip == nil {
		return compiledPattern{}, fmt.Errorf("invalid ip, cidr or range")
	}
	return compiledPattern{
		ipFast:  ipFastExact,
		ipExact: ip,
	}, nil
}

func sameIPFamily(a, b net.IP) bool {
	a4, b4 := a.To4() != nil, b.To4() != nil
	return a4 == b4
}

// DstIPForMatch — первый IP для acl dst (см. DstIPsForMatch).
func DstIPForMatch(f RequestFields) net.IP {
	ips := DstIPsForMatch(f)
	if len(ips) == 0 {
		return nil
	}
	return ips[0]
}

// ParseClientIP извлекает IP из RemoteAddr (host:port или IP).
func ParseClientIP(remoteAddr string) net.IP {
	remoteAddr = strings.TrimSpace(remoteAddr)
	if remoteAddr == "" {
		return nil
	}
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	return net.ParseIP(host)
}
