package auth

import (
	"context"
	"log"
	stdhttp "net/http"
	"strings"
	"time"
)

// AuthFailLogger пишет в журнал неудачную proxy-auth (без import accesslog в пакете auth).
type AuthFailLogger func(ctx context.Context, r *stdhttp.Request, spend time.Duration)

// Gate проверяет Proxy-Authorization до обработки запроса.
type Gate struct {
	Realm       string
	Auth        Authenticator
	Cache       *AuthCache
	Backend     string
	Log         *log.Logger
	LogAuthFail AuthFailLogger
}

// Require проверяет учётные данные. При отказе пишет 407 и возвращает ok=false.
func (g *Gate) Require(w stdhttp.ResponseWriter, r *stdhttp.Request) (context.Context, bool) {
	if g == nil || g.Auth == nil {
		return r.Context(), true
	}
	started := time.Now()
	fail := func() (context.Context, bool) {
		if g.LogAuthFail != nil {
			g.LogAuthFail(r.Context(), r, time.Since(started))
		}
		return r.Context(), false
	}

	realm := g.Realm
	if realm == "" {
		realm = "oktopus"
	}

	user, pass, ok := ProxyBasicCredentials(r)
	if !ok {
		challengeProxyBasic(w, realm)
		return fail()
	}

	if g.Cache != nil {
		if id, hit := g.Cache.Get(user, pass); hit {
			return WithIdentity(r.Context(), id), true
		}
	}

	id, err := g.Auth.Authenticate(r.Context(), user, pass)
	if err != nil {
		if IsInvalidCredentials(err) {
			challengeProxyBasic(w, realm)
			return fail()
		}
		stdhttp.Error(w, "Proxy Authentication Error", stdhttp.StatusBadGateway)
		return fail()
	}

	if g.Cache != nil {
		g.Cache.Set(user, pass, id)
		g.logLDAP("ldap auth: user=%q groups=%v saved to cache (ttl=%s)", id.Username, id.Groups, g.Cache.TTL())
	}

	return WithIdentity(r.Context(), id), true
}

func (g *Gate) logLDAP(format string, args ...any) {
	if g == nil || g.Log == nil {
		return
	}
	if strings.ToLower(g.Backend) != "ldap" {
		return
	}
	g.Log.Printf(format, args...)
}

func challengeProxyBasic(w stdhttp.ResponseWriter, realm string) {
	w.Header().Set("Proxy-Authenticate", `Basic realm="`+realm+`"`)
	w.WriteHeader(stdhttp.StatusProxyAuthRequired)
}
