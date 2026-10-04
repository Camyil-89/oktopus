package domain

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("proxy instance not found")
	ErrListenTaken   = errors.New("listen address already used")
	ErrNameTaken     = errors.New("instance name already used")
)

type Instance struct {
	ID                  uuid.UUID
	Name                string
	Enabled             bool
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
	LDAPBindPassword    string
	SortOrder           int
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func ConfigDir(id uuid.UUID) string {
	return filepath.Join("config", id.String())
}

func DefaultCAPaths(id uuid.UUID) (cert, key string) {
	dir := ConfigDir(id)
	return filepath.Join(dir, "ca.crt"), filepath.Join(dir, "ca.key")
}

func DefaultInstance(id uuid.UUID, name string, listen string, sortOrder int) Instance {
	cert, key := DefaultCAPaths(id)
	return Instance{
		ID:                  id,
		Name:                name,
		Enabled:             false,
		Listen:              listen,
		ConnectMode:         "tunnel",
		CACertPath:          cert,
		CAKeyPath:           key,
		AuthEnabled:         true,
		AuthStaticUsers:     "",
		AuthRealm:           "oktopus",
		AuthBackend:         "ldap",
		AuthCacheTTLMinutes: 5,
		LDAPURL:             "ldap://127.0.0.1:1389",
		LDAPBaseDN:          "dc=oktopus,dc=dev",
		LDAPBindDN:          "cn=admin,dc=oktopus,dc=dev",
		LDAPBindPassword:    "admin_dev",
		SortOrder:           sortOrder,
	}
}
