package auth

import (
	"fmt"
	"log"
	"strings"

	"oktopus/internal/proxy/config"
)

// NewGateFromConfig строит Gate по настройкам. Enabled=false → nil gate (auth выключена).
// sharedCache — общий кеш менеджера прокси; при nil создаётся новый экземпляр.
func NewGateFromConfig(cfg config.AuthConfig, logger *log.Logger, sharedCache *AuthCache) (*Gate, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	backend := strings.ToLower(strings.TrimSpace(cfg.Backend))
	if backend == "" {
		backend = "static"
	}

	var authenticator Authenticator
	switch backend {
	case "static":
		var st *Static
		var err error
		switch {
		case strings.TrimSpace(cfg.StaticUsers) != "":
			st, err = ParseStaticUsers(cfg.StaticUsers)
		case cfg.StaticFile != "":
			st, err = LoadStaticFile(cfg.StaticFile)
		default:
			return nil, fmt.Errorf("auth backend static requires users")
		}
		if err != nil {
			return nil, err
		}
		authenticator = st
	case "ldap":
		ld := cfg.LDAP
		if ld.URL == "" || ld.BindDN == "" {
			return nil, fmt.Errorf("auth backend ldap requires LDAP URL and BindDN")
		}
		authenticator = &LDAP{
			URL:          ld.URL,
			BaseDN:       ld.BaseDN,
			BindDN:       ld.BindDN,
			BindPassword: ld.BindPassword,
		}
	default:
		return nil, fmt.Errorf("unknown auth backend %q (supported: static, ldap)", cfg.Backend)
	}

	realm := cfg.Realm
	if realm == "" {
		realm = "oktopus"
	}

	gate := &Gate{
		Realm:   realm,
		Auth:    authenticator,
		Backend: backend,
		Log:     logger,
	}
	if cfg.CacheTTL > 0 {
		if sharedCache != nil {
			gate.Cache = sharedCache
		} else {
			gate.Cache = NewAuthCache(cfg.CacheTTL)
		}
	}
	return gate, nil
}
