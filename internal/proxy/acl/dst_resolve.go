package acl

import (
	"context"
	"net"
	"strings"
	"time"
)

const dstLookupTimeout = 5 * time.Second

// dstLookup resolves a hostname for acl dst (Squid-style: hostname → IP before dst match).
var dstLookup func(context.Context, string) ([]net.IP, error)

// SetDstLookupForTest подменяет DNS для unit-тестов (только в пакете acl).
func SetDstLookupForTest(fn func(context.Context, string) ([]net.IP, error)) {
	if fn == nil {
		dstLookup = defaultDstLookup
		return
	}
	dstLookup = fn
}

func defaultDstLookup(ctx context.Context, host string) ([]net.IP, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, dstLookupTimeout)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addrs))
	for _, a := range addrs {
		if ipValid(a.IP) {
			ips = append(ips, a.IP)
		}
	}
	return ips, nil
}

func init() {
	dstLookup = defaultDstLookup
}

// EnrichRequestFieldsDst заполняет DstResolved для hostname в SNI (не для литерального IP).
func EnrichRequestFieldsDst(ctx context.Context, f RequestFields) RequestFields {
	if ipValid(f.DstIP) || len(f.DstResolved) > 0 {
		return f
	}
	host := hostnameFromSNI(f.SNI)
	if host == "" || net.ParseIP(host) != nil {
		return f
	}
	ips, err := dstLookup(ctx, host)
	if err != nil || len(ips) == 0 {
		return f
	}
	f.DstResolved = ips
	return f
}

func hostnameFromSNI(sni string) string {
	host := strings.TrimSpace(sni)
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.Trim(host, "[]")
}

// DstIPsForMatch — IP для acl dst: явное поле, резолв, или литерал в SNI.
func DstIPsForMatch(f RequestFields) []net.IP {
	if ipValid(f.DstIP) {
		return []net.IP{f.DstIP}
	}
	if len(f.DstResolved) > 0 {
		return f.DstResolved
	}
	ip := parseIPLiteral(hostnameFromSNI(f.SNI))
	if ipValid(ip) {
		return []net.IP{ip}
	}
	return nil
}

func parseIPLiteral(host string) net.IP {
	if host == "" {
		return nil
	}
	return net.ParseIP(host)
}

func dstPatternMatches(p compiledPattern, f RequestFields) bool {
	for _, ip := range DstIPsForMatch(f) {
		if ipPatternMatches(p, ip) {
			return true
		}
	}
	return false
}

func needsDstResolve(skip map[RuleType]bool) bool {
	if skip == nil {
		return true
	}
	return !skip[RuleDST]
}
