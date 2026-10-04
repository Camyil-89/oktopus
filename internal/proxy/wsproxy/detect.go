package wsproxy

import (
	stdhttp "net/http"
	"strings"
)

// IsWebSocketUpgrade — HTTP GET с Connection: Upgrade и Upgrade: websocket (RFC 6455).
func IsWebSocketUpgrade(req *stdhttp.Request) bool {
	if req == nil || req.Method != stdhttp.MethodGet {
		return false
	}
	if !headerTokenListContains(req.Header, "Connection", "upgrade") {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(req.Header.Get("Upgrade")), "websocket")
}

func headerTokenListContains(h stdhttp.Header, key, want string) bool {
	for _, part := range strings.Split(h.Get(key), ",") {
		if strings.EqualFold(strings.TrimSpace(part), want) {
			return true
		}
	}
	return false
}
