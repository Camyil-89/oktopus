package setup

import (
	"fmt"
	"strings"
)

// FormatPOCHTTPResponse — краткая строка для логов PoC (без дампа HTML Forbidden).
func FormatPOCHTTPResponse(status int, body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Sprintf("HTTP %d", status)
	}
	if strings.Contains(body, "SECRET_INTERNAL_HIT") {
		return fmt.Sprintf("HTTP %d — internal hit", status)
	}
	if status == 403 || strings.Contains(body, "<!DOCTYPE") || strings.Contains(body, "errorDetail") {
		if msg := gatewayForbiddenDetail(body); msg != "" {
			return fmt.Sprintf("HTTP %d — %s", status, msg)
		}
		return fmt.Sprintf("HTTP %d — forbidden", status)
	}
	const max = 120
	if len(body) > max {
		return fmt.Sprintf("HTTP %d — %s…", status, body[:max])
	}
	return fmt.Sprintf("HTTP %d — %s", status, body)
}

// POCResponseSnippet — фрагмент тела для bypass; пустая строка, если достаточно статуса (HTML gateway).
func POCResponseSnippet(status int, body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	if status == 403 || strings.Contains(body, "<!DOCTYPE") || strings.Contains(body, "errorDetail") {
		return ""
	}
	if strings.Contains(body, "SECRET_INTERNAL_HIT") {
		return "SECRET_INTERNAL_HIT"
	}
	const max = 120
	if len(body) > max {
		return body[:max] + "…"
	}
	return body
}

func gatewayForbiddenDetail(html string) string {
	const marker = `id="errorDetail">`
	i := strings.Index(html, marker)
	if i < 0 {
		return ""
	}
	rest := html[i+len(marker):]
	if j := strings.Index(rest, "</div>"); j > 0 {
		rest = rest[:j]
	}
	rest = strings.TrimSpace(rest)
	const max = 200
	if len(rest) > max {
		rest = rest[:max] + "…"
	}
	return rest
}
