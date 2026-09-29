package squid

import (
	"strings"
	"unicode"
)

type fieldSpan struct {
	text  string
	start int // byte offset in full line (0-based)
	end   int
}

func lineTrimOffset(raw string) int {
	return len(raw) - len(strings.TrimLeft(raw, " \t"))
}

func splitFieldSpans(line string) []fieldSpan {
	var out []fieldSpan
	var cur strings.Builder
	start := -1
	inQuote := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '"' {
			if cur.Len() == 0 {
				start = i
			}
			inQuote = !inQuote
			cur.WriteByte(c)
			continue
		}
		if !inQuote && unicode.IsSpace(rune(c)) {
			if cur.Len() > 0 {
				out = append(out, fieldSpan{
					text:  cur.String(),
					start: start,
					end:   i,
				})
				cur.Reset()
				start = -1
			}
			continue
		}
		if cur.Len() == 0 {
			start = i
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		out = append(out, fieldSpan{
			text:  cur.String(),
			start: start,
			end:   len(line),
		})
	}
	return out
}

func spanColumns(raw string, sp fieldSpan) (col, endCol int) {
	trim := lineTrimOffset(raw)
	// 1-based columns in editor
	col = sp.start - trim + 1
	endCol = sp.end - trim + 1
	if col < 1 {
		col = 1
	}
	if endCol < col {
		endCol = col
	}
	return col, endCol
}
