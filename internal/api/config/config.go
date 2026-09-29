package config

import (
	"os"
	"strings"
)

const (
	envListen          = "OKTOPUS_API_LISTEN"
	envJWTSecret       = "OKTOPUS_JWT_SECRET"
	envCORSOrigins     = "OKTOPUS_CORS_ORIGINS"
	envCookieSecure    = "OKTOPUS_COOKIE_SECURE"
	envAdminUsername   = "OKTOPUS_ADMIN_USERNAME"
	defaultListen     = "127.0.0.1:8000"
	defaultCORSOrigin = "http://localhost:3000"
	defaultAdminUser   = "admin"
)

// Config HTTP API.
type Config struct {
	Listen         string
	JWTSecret      string
	CORSOrigins    []string
	CookieSecure   bool
	AdminUsername  string
}

// Load читает настройки API: env, при отсутствии jwt_secret — файл config/oktopus.local.json
// (создаётся с новым секретом при первом запуске).
func Load() (Config, error) {
	jwtSecret, err := resolveJWTSecret()
	if err != nil {
		return Config{}, err
	}

	origins := strings.Split(envOr(envCORSOrigins, defaultCORSOrigin), ",")
	trimmed := make([]string, 0, len(origins))
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o != "" {
			trimmed = append(trimmed, o)
		}
	}
	return Config{
		Listen:        envOr(envListen, defaultListen),
		JWTSecret:     jwtSecret,
		CORSOrigins:   trimmed,
		CookieSecure:  os.Getenv(envCookieSecure) == "true" || os.Getenv(envCookieSecure) == "1",
		AdminUsername: envOr(envAdminUsername, defaultAdminUser),
	}, nil
}

// FromEnv — алиас Load; при ошибке конфига возвращает нулевой Config (используйте Load в cmd).
func FromEnv() Config {
	cfg, err := Load()
	if err != nil {
		return Config{}
	}
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
