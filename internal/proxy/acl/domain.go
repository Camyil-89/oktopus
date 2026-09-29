package acl

import (
	"strings"
	"unicode"

	"golang.org/x/net/idna"
)

type sniFastKind int

const (
	sniFastNone sniFastKind = iota
	sniFastExact
	sniFastSuffix
	sniFastWildcardSuffix
)

type domainNode struct {
	suffixRule  int
	wildRule    int
	suffixDepth int
	children    map[string]*domainNode
}

type sniIndex struct {
	exact map[string]int
	root  domainNode
}

func setEarliest(m map[string]int, key string, ruleIdx int) {
	key = internCompileLabel(key)
	if prev, ok := m[key]; !ok || ruleIdx < prev {
		m[key] = ruleIdx
	}
}

func (idx *sniIndex) insertSuffix(domain string, ruleIdx int) {
	labels := domainLabels(domain)
	if len(labels) == 0 {
		return
	}
	node := &idx.root
	for i := len(labels) - 1; i >= 0; i-- {
		node = node.child(labels[i])
	}
	if node.suffixRule < 0 || ruleIdx < node.suffixRule {
		node.suffixRule = ruleIdx
		node.suffixDepth = len(labels)
	}
}

func (idx *sniIndex) insertWildcard(domain string, ruleIdx int) {
	labels := domainLabels(domain)
	if len(labels) == 0 {
		return
	}
	node := &idx.root
	for i := len(labels) - 1; i >= 0; i-- {
		node = node.child(labels[i])
	}
	if node.wildRule < 0 || ruleIdx < node.wildRule {
		node.wildRule = ruleIdx
		node.suffixDepth = len(labels)
	}
}

func (n *domainNode) child(label string) *domainNode {
	label = internCompileLabel(label)
	if n.children == nil {
		n.children = make(map[string]*domainNode)
	}
	c := n.children[label]
	if c == nil {
		c = &domainNode{suffixRule: -1, wildRule: -1}
		n.children[label] = c
	}
	return c
}

func (idx *sniIndex) hasMatch(sniLower string) bool {
	if idx == nil {
		return false
	}
	if _, ok := idx.exact[sniLower]; ok {
		return true
	}
	labels := domainLabels(sniLower)
	if len(labels) == 0 {
		return false
	}
	node := &idx.root
	for i := len(labels) - 1; i >= 0; i-- {
		if node.children == nil {
			break
		}
		child := node.children[labels[i]]
		if child == nil {
			break
		}
		node = child
		if node.suffixRule >= 0 {
			return true
		}
		if node.wildRule >= 0 && len(labels) > node.suffixDepth {
			return true
		}
	}
	return false
}

func (idx *sniIndex) match(sniLower string) []int {
	if idx == nil {
		return nil
	}
	var out []int
	if ruleIdx, ok := idx.exact[sniLower]; ok {
		out = append(out, ruleIdx)
	}
	labels := domainLabels(sniLower)
	if len(labels) == 0 {
		return out
	}
	node := &idx.root
	for i := len(labels) - 1; i >= 0; i-- {
		if node.children == nil {
			break
		}
		child := node.children[labels[i]]
		if child == nil {
			break
		}
		node = child
		if node.suffixRule >= 0 {
			out = append(out, node.suffixRule)
		}
		if node.wildRule >= 0 && len(labels) > node.suffixDepth {
			out = append(out, node.wildRule)
		}
	}
	return out
}

func domainPatternMatches(kind sniFastKind, patternHost, sniLower string) bool {
	if patternHost == "" || sniLower == "" {
		return false
	}
	switch kind {
	case sniFastExact:
		return sniLower == patternHost
	case sniFastSuffix:
		return sniLower == patternHost || strings.HasSuffix(sniLower, "."+patternHost)
	case sniFastWildcardSuffix:
		if sniLower == patternHost {
			return false
		}
		return strings.HasSuffix(sniLower, "."+patternHost)
	default:
		return false
	}
}

func domainLabels(host string) []string {
	host = strings.Trim(host, ".")
	if host == "" {
		return nil
	}
	return strings.Split(host, ".")
}

// parseDomainPattern распознаёт литералы *.domain, .domain, domain (без regex-синтаксиса).
func parseDomainPattern(pat string) (sniFastKind, string, bool) {
	pat = strings.TrimSpace(pat)
	if pat == "" {
		return sniFastNone, "", false
	}
	if strings.HasPrefix(pat, "*.") {
		dom := strings.Trim(strings.TrimPrefix(pat, "*."), ".")
		if !isDomainLiteral(dom) {
			return sniFastNone, "", false
		}
		return sniFastWildcardSuffix, normalizeHostForMatch(dom), true
	}
	if strings.HasPrefix(pat, ".") {
		dom := strings.Trim(strings.TrimPrefix(pat, "."), ".")
		if !isDomainLiteral(dom) {
			return sniFastNone, "", false
		}
		return sniFastSuffix, normalizeHostForMatch(dom), true
	}
	if isDomainLiteral(pat) {
		return sniFastExact, normalizeHostForMatch(pat), true
	}
	return sniFastNone, "", false
}

// normalizeHostForMatch приводит хост к ASCII (punycode) и нижнему регистру для сопоставления IDN.
func normalizeHostForMatch(host string) string {
	host = strings.TrimSpace(host)
	host = strings.Trim(host, ".")
	if host == "" {
		return ""
	}
	if hostIsASCIIDomain(host) {
		return asciiDomainLower(host)
	}
	ascii, err := idna.ToASCII(host)
	if err != nil {
		return strings.ToLower(host)
	}
	return strings.ToLower(ascii)
}

func hostIsASCIIDomain(host string) bool {
	for i := 0; i < len(host); i++ {
		c := host[i]
		if c >= 'A' && c <= 'Z' {
			return false
		}
		if c >= 'a' && c <= 'z' {
			continue
		}
		if c >= '0' && c <= '9' || c == '.' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func asciiDomainLower(host string) string {
	needsLower := false
	for i := 0; i < len(host); i++ {
		if host[i] >= 'A' && host[i] <= 'Z' {
			needsLower = true
			break
		}
	}
	if !needsLower {
		return host
	}
	var b []byte
	for i := 0; i < len(host); i++ {
		c := host[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b = append(b, c)
	}
	return string(b)
}

// isDomainLiteral — литерал домена без regex-синтаксиса (ASCII, IDN/кириллица и т.д.).
func isDomainLiteral(s string) bool {
	if s == "" || strings.HasPrefix(s, ".") || strings.HasSuffix(s, ".") || strings.Contains(s, "..") {
		return false
	}
	labels := strings.Split(s, ".")
	for _, lab := range labels {
		if lab == "" {
			return false
		}
		if strings.HasPrefix(lab, "-") || strings.HasSuffix(lab, "-") {
			return false
		}
		for _, r := range lab {
			if r == '-' || r == '_' {
				continue
			}
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				continue
			}
			return false
		}
	}
	return true
}
