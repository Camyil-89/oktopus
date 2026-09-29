package auth

import (
	"encoding/base64"
	stdhttp "net/http"
	"strings"
)

// ProxyBasicCredentials читает логин и пароль из Proxy-Authorization: Basic.
func ProxyBasicCredentials(r *stdhttp.Request) (username, password string, ok bool) {
	h := r.Header.Get("Proxy-Authorization")
	if !strings.HasPrefix(h, "Basic ") {
		return "", "", false
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(h[len("Basic "):]))
	if err != nil {
		return "", "", false
	}
	user, pass, found := strings.Cut(string(raw), ":")
	if !found || user == "" {
		return "", "", false
	}
	return user, pass, true
}
