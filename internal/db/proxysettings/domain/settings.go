package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("proxy settings not found")

const (
	DefaultCACertPath = "config/ca.crt"
	DefaultCAKeyPath  = "config/ca.key"
)

// Settings — конфигурация HTTP-прокси (кроме ACL).
type Settings struct {
	ID                  uuid.UUID
	ProxyEnabled        bool
	Listen              string
	ConnectMode         string
	CACertPath          string
	CAKeyPath           string
	AuthEnabled         bool
	AuthStaticUsers     string
	AuthRealm           string
	AuthBackend         string
	AuthCacheTTLMinutes int64
	LDAPURL             string
	LDAPBaseDN          string
	LDAPBindDN          string
	LDAPBindPassword         string
	AccessLogRetentionDays   int32 // 0 — без ограничения
	CreatedAt                time.Time
	UpdatedAt           time.Time
}

func DefaultSettings(id uuid.UUID) Settings {
	return Settings{
		ID:                  id,
		ProxyEnabled:        false,
		Listen:              "127.0.0.1:8080",
		ConnectMode:         "tunnel",
		CACertPath:          DefaultCACertPath,
		CAKeyPath:           DefaultCAKeyPath,
		AuthEnabled:         true,
		AuthStaticUsers:     "",
		AuthRealm:           "oktopus",
		AuthBackend:         "ldap",
		AuthCacheTTLMinutes: 5,
		LDAPURL:             "ldap://127.0.0.1:1389",
		LDAPBaseDN:          "dc=oktopus,dc=dev",
		LDAPBindDN:          "cn=admin,dc=oktopus,dc=dev",
		LDAPBindPassword:       "admin_dev",
		AccessLogRetentionDays: 3,
	}
}

func (s Settings) NormalizeCAPaths() Settings {
	s.CACertPath = DefaultCACertPath
	s.CAKeyPath = DefaultCAKeyPath
	return s
}
