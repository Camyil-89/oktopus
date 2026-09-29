package http

import (
	stdhttp "net/http"
	"strings"
)

var hopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

// StripHopByHopHeaders удаляет hop-by-hop заголовки (RFC 7230).
func StripHopByHopHeaders(h stdhttp.Header) {
	removeHopByHopHeaders(h)
}

func removeHopByHopHeaders(h stdhttp.Header) {
	for _, k := range hopByHopHeaders {
		h.Del(k)
	}
	if c := h.Get("Connection"); c != "" {
		for _, part := range strings.Split(c, ",") {
			h.Del(strings.TrimSpace(part))
		}
	}
}

// CopyHeaders копирует заголовки ответа origin клиенту.
func CopyHeaders(dst, src stdhttp.Header) {
	copyHeader(dst, src)
}

func copyHeader(dst, src stdhttp.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
