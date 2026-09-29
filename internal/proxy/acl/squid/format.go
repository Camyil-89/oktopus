package squid

import (
	"strings"
)

// FormatAccessDirective — текст строки политики (http_access allow all …).
func FormatAccessDirective(kind string, act string, clauses []AccessClause) string {
	kind = strings.TrimSpace(kind)
	act = strings.TrimSpace(act)
	var b strings.Builder
	b.WriteString(kind)
	if act != "" {
		b.WriteByte(' ')
		b.WriteString(act)
	}
	for _, cl := range clauses {
		b.WriteByte(' ')
		if cl.Negated {
			b.WriteByte('!')
		}
		b.WriteString(cl.Name)
	}
	return b.String()
}
