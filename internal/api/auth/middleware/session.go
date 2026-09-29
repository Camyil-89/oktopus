package middleware

import (
	"net/http"

	"oktopus/internal/auth"
)

// ClaimsFromRequest извлекает JWT из cookie сессии.
func ClaimsFromRequest(r *http.Request, secret []byte) (auth.Claims, bool) {
	raw, ok := auth.SessionTokenFromRequest(r)
	if !ok {
		return auth.Claims{}, false
	}
	claims, err := auth.ParseToken(secret, raw)
	if err != nil {
		return auth.Claims{}, false
	}
	return claims, true
}

