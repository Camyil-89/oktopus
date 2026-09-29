package http

import (
	"net/url"
	"strings"
)

// ParseSearchQuery — типичные query-параметры (q, query, text, …) на любом URL;
// для Lua/observe. Для журнала access log используйте ParseSearchRequest.
func ParseSearchQuery(u *url.URL) string {
	if u == nil {
		return ""
	}
	q := u.Query()
	for _, key := range []string{"q", "query", "text", "p", "wd"} {
		if v := q.Get(key); v != "" {
			return v
		}
	}
	return ""
}

// ParseSearchRequest — известная поисковая система и текст запроса, иначе ("", "").
func ParseSearchRequest(u *url.URL) (engine, query string) {
	if u == nil {
		return "", ""
	}
	engine = detectSearchEngine(u)
	if engine == "" {
		return "", ""
	}
	query = strings.TrimSpace(searchQueryForEngine(engine, u))
	if query == "" {
		return "", ""
	}
	return engine, query
}

func detectSearchEngine(u *url.URL) string {
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	host = strings.TrimPrefix(host, "www.")
	if host == "" {
		return ""
	}
	path := strings.ToLower(u.EscapedPath())
	if path == "" {
		path = "/"
	}

	switch {
	case strings.Contains(host, "google."):
		if !googleSearchPath(path) {
			return ""
		}
		return "google"
	case strings.Contains(host, "yandex.") || host == "ya.ru":
		if !strings.Contains(path, "/search") {
			return ""
		}
		return "yandex"
	case strings.HasSuffix(host, "bing.com"):
		if !bingSearchPath(path) {
			return ""
		}
		return "bing"
	case strings.Contains(host, "duckduckgo.com"):
		return "duckduckgo"
	case strings.Contains(host, "yahoo.com"):
		if !strings.HasPrefix(path, "/search") {
			return ""
		}
		return "yahoo"
	case strings.Contains(host, "mail.ru") && strings.Contains(path, "search"):
		return "mail.ru"
	default:
		return ""
	}
}

func googleSearchPath(path string) bool {
	return strings.HasPrefix(path, "/search") ||
		strings.HasPrefix(path, "/url") ||
		strings.HasPrefix(path, "/imgres")
}

func bingSearchPath(path string) bool {
	return strings.HasPrefix(path, "/search") || path == "/"
}

func searchQueryForEngine(engine string, u *url.URL) string {
	q := u.Query()
	switch engine {
	case "google", "bing", "duckduckgo", "mail.ru":
		return q.Get("q")
	case "yandex":
		if v := q.Get("text"); v != "" {
			return v
		}
		return q.Get("query")
	case "yahoo":
		return q.Get("p")
	default:
		return ""
	}
}
