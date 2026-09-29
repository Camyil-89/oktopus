package auth

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

const ldapUserFilter = "(uid=%s)"

// LDAP — проверка Basic через bind пользователя и чтение групп (member=).
type LDAP struct {
	URL          string
	BaseDN       string
	BindDN       string
	BindPassword string
	Timeout      time.Duration
}

func (l *LDAP) Authenticate(_ context.Context, username, password string) (Identity, error) {
	if l.Timeout <= 0 {
		l.Timeout = 10 * time.Second
	}
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return Identity{}, ErrInvalidCredentials{}
	}

	userDN, err := l.findUserDN(username)
	if err != nil {
		if IsInvalidCredentials(err) {
			return Identity{}, err
		}
		return Identity{}, fmt.Errorf("ldap find user: %w", err)
	}
	if err := l.bindUser(userDN, password); err != nil {
		if IsInvalidCredentials(err) {
			return Identity{}, err
		}
		return Identity{}, fmt.Errorf("ldap bind user: %w", err)
	}

	groups, err := l.loadGroups(userDN)
	if err != nil {
		return Identity{}, fmt.Errorf("ldap groups: %w", err)
	}

	return Identity{Username: username, Groups: groups}, nil
}

func (l *LDAP) findUserDN(username string) (string, error) {
	conn, err := l.dial()
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if err := conn.Bind(l.BindDN, l.BindPassword); err != nil {
		return "", fmt.Errorf("service bind: %w", err)
	}

	filter := strings.Replace(ldapUserFilter, "%s", ldap.EscapeFilter(username), 1)
	req := ldap.NewSearchRequest(
		l.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, 0, false,
		filter,
		[]string{"dn"},
		nil,
	)
	res, err := conn.Search(req)
	if err != nil {
		return "", err
	}
	if len(res.Entries) == 0 {
		return "", ErrInvalidCredentials{}
	}
	return res.Entries[0].DN, nil
}

func (l *LDAP) bindUser(userDN, password string) error {
	conn, err := l.dial()
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.Bind(userDN, password); err != nil {
		if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
			return ErrInvalidCredentials{}
		}
		return err
	}
	return nil
}

func (l *LDAP) loadGroups(userDN string) ([]string, error) {
	conn, err := l.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := conn.Bind(l.BindDN, l.BindPassword); err != nil {
		return nil, err
	}

	filter := fmt.Sprintf("(member=%s)", ldap.EscapeFilter(userDN))
	req := ldap.NewSearchRequest(
		l.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		filter,
		[]string{"cn"},
		nil,
	)
	res, err := conn.Search(req)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(res.Entries))
	for _, e := range res.Entries {
		if cn := e.GetAttributeValue("cn"); cn != "" {
			out = append(out, cn)
		}
	}
	return out, nil
}

func (l *LDAP) dial() (*ldap.Conn, error) {
	if l.URL == "" {
		return nil, fmt.Errorf("ldap: empty URL")
	}
	return ldap.DialURL(l.URL, ldap.DialWithDialer(&net.Dialer{Timeout: l.Timeout}))
}
