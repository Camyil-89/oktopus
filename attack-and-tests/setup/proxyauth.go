package setup

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Dev LDAP (docker-compose.dev.yml, deploy/openldap).
const (
	LDAPDevURL          = "ldap://127.0.0.1:1389"
	LDAPDevBaseDN       = "dc=oktopus,dc=dev"
	LDAPDevBindDN       = "cn=admin,dc=oktopus,dc=dev"
	LDAPDevBindPassword = "admin_dev"
	LDAPDevUser         = "user"
	LDAPDevPass         = "user"
	LDAPDevUser2        = "user2"
	LDAPDevPass2        = "user2"
	LDAPDevGroupAllow   = "allow_all"
	LDAPDevGroupDeny    = "deny_all"
)

const proxyAuthProbeDest = "example.com:443"

// CheckLDAPDevReachable — TCP до dev OpenLDAP (docker-compose.dev.yml).
func CheckLDAPDevReachable() error {
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.Dial("tcp", "127.0.0.1:1389")
	if err != nil {
		return fmt.Errorf("ldap %s not reachable (%v); start OpenLDAP (see docker-compose.dev.yml)", LDAPDevURL, err)
	}
	conn.Close()
	return nil
}

// ProxyAuthCONNECTStatus — HTTP-код ответа на CONNECT (407 = отказ proxy-auth).
func ProxyAuthCONNECTStatus(proxyAddr, user, pass string) (int, error) {
	status, conn, err := dialCONNECTProbe(proxyAddr, proxyAuthProbeDest, user, pass)
	if conn != nil {
		conn.Close()
	}
	if err != nil && status == 0 {
		return 0, err
	}
	return status, nil
}

// EnableProxyStaticAuth включает прокси с backend static и заданным списком login:pass.
func (c *Client) EnableProxyStaticAuth(connectMode, staticUsers string) (listen string, err error) {
	connectMode, err = normalizeConnectMode(connectMode)
	if err != nil {
		return "", err
	}
	staticUsers = strings.TrimSpace(staticUsers)
	if staticUsers == "" {
		return "", fmt.Errorf("static users list empty")
	}
	return c.patchProxyAuthAndWait(map[string]interface{}{
		"proxy_enabled":     true,
		"connect_mode":      connectMode,
		"auth_enabled":      true,
		"auth_backend":      "static",
		"auth_static_users": staticUsers,
	})
}

// EnableProxyLDAPAuth включает прокси с backend ldap (dev OpenLDAP).
func (c *Client) EnableProxyLDAPAuth(connectMode string) (listen string, err error) {
	connectMode, err = normalizeConnectMode(connectMode)
	if err != nil {
		return "", err
	}
	return c.patchProxyAuthAndWait(map[string]interface{}{
		"proxy_enabled":      true,
		"connect_mode":       connectMode,
		"auth_enabled":       true,
		"auth_backend":       "ldap",
		"ldap_url":           LDAPDevURL,
		"ldap_base_dn":       LDAPDevBaseDN,
		"ldap_bind_dn":       LDAPDevBindDN,
		"ldap_bind_password": LDAPDevBindPassword,
	})
}

func normalizeConnectMode(connectMode string) (string, error) {
	connectMode = strings.TrimSpace(strings.ToLower(connectMode))
	if connectMode != "mitm" && connectMode != "tunnel" {
		return "", fmt.Errorf("connect_mode must be mitm or tunnel, got %q", connectMode)
	}
	return connectMode, nil
}

func (c *Client) patchProxyAuthAndWait(body map[string]interface{}) (listen string, err error) {
	if err := c.patchJSON("/api/proxy/settings", body); err != nil {
		return "", err
	}
	return c.waitProxyActive(30 * time.Second)
}

func (c *Client) waitProxyActive(timeout time.Duration) (listen string, err error) {
	deadline := time.Now().Add(timeout)
	for {
		st, err := c.ProxyStatus()
		if err != nil {
			return "", err
		}
		if st.ProxyActive {
			listen = strings.TrimSpace(st.Listen)
			if listen == "" {
				return "", fmt.Errorf("proxy listen empty")
			}
			return listen, nil
		}
		if st.ProxyStartError != "" {
			return "", fmt.Errorf("proxy not active: %s", st.ProxyStartError)
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("timeout waiting for proxy active")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// ProxyAuthDenied — ожидаемый отказ proxy-auth.
func ProxyAuthDenied(status int) bool {
	return status == http.StatusProxyAuthRequired
}

// ProxyAuthAccepted — учётные данные приняты (CONNECT установлен).
func ProxyAuthAccepted(status int) bool {
	return status == http.StatusOK
}
