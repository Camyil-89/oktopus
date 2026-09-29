package config

import (
	"time"
)
// AuthConfig — HTTP Proxy Authentication (Basic); backend: static | ldap.
type AuthConfig struct {
	Enabled    bool
	Realm      string
	Backend      string // "static" | "ldap"
	StaticUsers  string // login:password построчно (из БД)
	StaticFile   string // CLI: путь к файлу
	CacheTTL     time.Duration // 0 = без кеша; иначе длительность (из минут в service)
	LDAP       LDAPConfig
}

// LDAPConfig — параметры dev/prod LDAP (backend ldap).
type LDAPConfig struct {
	URL          string
	BaseDN       string
	BindDN       string
	BindPassword string
}

// Config — общие параметры прокси-сервера.
type Config struct {
	Listen            string
	Connect           ConnectMode
	Auth              AuthConfig
	OutboundTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	IdleTimeout       time.Duration
	CACertPath        string
	CAKeyPath         string
}

// ConnectMode — как обрабатывать HTTPS (метод CONNECT).
type ConnectMode string

const (
	ConnectTunnel ConnectMode = "tunnel"
	ConnectMITM   ConnectMode = "mitm"
)

// WithDefaults заполняет пустые поля значениями по умолчанию.
func (c Config) WithDefaults() Config {
	if c.Listen == "" {
		c.Listen = "127.0.0.1:8080"
	}
	if c.Connect == "" {
		c.Connect = ConnectTunnel
	}
	if c.OutboundTimeout == 0 {
		c.OutboundTimeout = 60 * time.Second
	}
	if c.ReadHeaderTimeout == 0 {
		c.ReadHeaderTimeout = 10 * time.Second
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = 120 * time.Second
	}
	if c.CACertPath == "" {
		c.CACertPath = "config/ca.crt"
	}
	if c.CAKeyPath == "" {
		c.CAKeyPath = "config/ca.key"
	}
	return c
}
