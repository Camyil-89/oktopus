package acl

import (
	"net"
	"net/url"
	"strings"
	"sync"
	"unicode"
)

const maxSNIRegexpSamples = 40

// SNIPatternRegexpSample — строка, скомпилированная как regexp (не fast index).
type SNIPatternRegexpSample struct {
	Line   string `json:"line"`
	Reason string `json:"reason"`
}

// SNIPatternIndexReport — итог компиляции SNI-строк (для API/диагностики).
type SNIPatternIndexReport struct {
	RegexpReasonCounts map[string]int      `json:"regexp_reason_counts"`
	RegexpSamples      []SNIPatternRegexpSample `json:"regexp_samples"`
}

type sniIndexCompileAcc struct {
	mu      sync.Mutex
	reasons map[string]int
	samples []SNIPatternRegexpSample
}

func newSNIIndexCompileAcc() *sniIndexCompileAcc {
	return &sniIndexCompileAcc{reasons: make(map[string]int)}
}

func (a *sniIndexCompileAcc) noteRegexp(line, reason string) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.reasons[reason]++
	if len(a.samples) >= maxSNIRegexpSamples {
		return
	}
	a.samples = append(a.samples, SNIPatternRegexpSample{Line: line, Reason: reason})
}

func (a *sniIndexCompileAcc) report() SNIPatternIndexReport {
	if a == nil {
		return SNIPatternIndexReport{}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	reasons := make(map[string]int, len(a.reasons))
	for k, v := range a.reasons {
		reasons[k] = v
	}
	return SNIPatternIndexReport{
		RegexpReasonCounts: reasons,
		RegexpSamples:      append([]SNIPatternRegexpSample(nil), a.samples...),
	}
}

// ClassifySNIPatternIndex — можно ли строку положить в SNI fast index (без regexp).
func ClassifySNIPatternIndex(line string) (indexed bool, reason string) {
	_, ok := compileSNIFastPattern(line)
	if ok {
		return true, ""
	}
	return false, sniRegexpFallbackReason(line)
}

func sniLineIndexCandidates(line string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(line)
	add(normalizeSNIPatternLine(line))
	return out
}

// normalizeSNIPatternLine — типичные обёртки списков (adblock, URL, порт, точка).
func normalizeSNIPatternLine(line string) string {
	s := strings.TrimSpace(line)
	for strings.HasPrefix(s, "||") {
		s = strings.TrimSpace(s[2:])
	}
	s = strings.TrimSuffix(s, "^")
	s = strings.TrimSuffix(s, "$")
	s = strings.Trim(s, "|")
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err == nil {
			if h := u.Hostname(); h != "" {
				s = h
			}
		}
	}
	if h, p, err := net.SplitHostPort(s); err == nil && p != "" {
		s = h
	}
	if i := strings.Index(s, "/"); i > 0 && !strings.Contains(s[:i], ":") {
		s = s[:i]
	}
	s = strings.Trim(s, ".")
	return strings.TrimSpace(s)
}

func compileSNIFastPattern(pat string) (compiledPattern, bool) {
	for _, cand := range sniLineIndexCandidates(pat) {
		if kind, host, ok := parseDomainPattern(cand); ok {
			return compiledPattern{sniFast: kind, sniHost: internCompileLabel(host)}, true
		}
		if host, ok := peelSimpleSNIRegex(cand); ok {
			return compiledPattern{sniFast: sniFastExact, sniHost: internCompileLabel(host)}, true
		}
	}
	if host, ok := peelSimpleSNIRegex(pat); ok {
		return compiledPattern{sniFast: sniFastExact, sniHost: internCompileLabel(host)}, true
	}
	return compiledPattern{}, false
}

// peelSimpleSNIRegex — «regex» вида (?i)^foo\.bar$ без alternation/классов → exact в SNI index.
func peelSimpleSNIRegex(pat string) (hostLower string, ok bool) {
	s := strings.TrimSpace(pat)
	if strings.HasPrefix(s, "(?i)") {
		s = strings.TrimSpace(s[4:])
	}
	if strings.HasPrefix(s, "^") {
		s = s[1:]
	}
	if strings.HasSuffix(s, "$") {
		s = s[:len(s)-1]
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' {
			if i+1 >= len(s) {
				return "", false
			}
			switch s[i+1] {
			case '.':
				b.WriteByte('.')
				i++
			default:
				return "", false
			}
			continue
		}
		if strings.ContainsRune("[](){}?+*|^$", rune(s[i])) {
			return "", false
		}
		b.WriteByte(s[i])
	}
	dom := b.String()
	if !isDomainLiteral(dom) {
		return "", false
	}
	return normalizeHostForMatch(dom), true
}

func sniRegexpFallbackReason(pat string) string {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return "empty"
	}
	if strings.ContainsAny(pat, "[](){}?+*\\|^$") {
		return "regex_syntax"
	}
	if strings.Contains(pat, "..") {
		return "invalid_domain"
	}
	if strings.Contains(pat, "_") {
		return "underscore"
	}
	if strings.Contains(pat, ":") {
		return "port_or_ipv6"
	}
	if strings.HasPrefix(pat, "/") || strings.Contains(pat, "/") {
		return "path_not_sni"
	}
	for _, r := range pat {
		if r == '.' || r == '-' || r == '*' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		return "invalid_char"
	}
	return "not_domain_literal"
}
