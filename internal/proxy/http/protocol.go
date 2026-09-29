package http

import stdhttp "net/http"

// StripHTTP3Hints убирает подсказки браузеру перейти на HTTP/3 (QUIC) в обход прокси.
func StripHTTP3Hints(h stdhttp.Header) {
	h.Del("Alt-Svc")
	h.Del("Alt-Svc-Clear")
	h.Del("HTTP2-Settings")
}
