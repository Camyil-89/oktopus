package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"oktopus/internal/proxy"
)

type proxyCLI struct {
	Listen       string
	Connect      string
	CACert       string
	CAKey        string
	AuthEnable   bool
	AuthFile     string
	AuthRealm    string
	AuthBackend  string
	AuthCacheTTL time.Duration
	LDAPURL      string
	LDAPBaseDN   string
	LDAPBindDN   string
	LDAPBindPass string
}

func registerProxyFlags(fs *flag.FlagSet, p *proxyCLI) {
	fs.StringVar(&p.Listen, "listen", "127.0.0.1:8080", "адрес прослушивания прокси")
	fs.StringVar(&p.Connect, "connect", "tunnel", "CONNECT: mitm | tunnel")
	fs.StringVar(&p.CACert, "ca-cert", "config/ca.crt", "корневой сертификат для mitm")
	fs.StringVar(&p.CAKey, "ca-key", "config/ca.key", "ключ CA для mitm")
	fs.BoolVar(&p.AuthEnable, "auth", true, "требовать HTTP Proxy Authentication (Basic)")
	fs.StringVar(&p.AuthFile, "auth-file", "config/proxy-users.txt", "файл учёток login:password (backend static)")
	fs.StringVar(&p.AuthRealm, "auth-realm", "oktopus", "realm в Proxy-Authenticate")
	fs.StringVar(&p.AuthBackend, "auth-backend", "ldap", "backend: static | ldap")
	fs.DurationVar(&p.AuthCacheTTL, "auth-cache-ttl", 5*time.Minute, "кеш успешной auth; 0=выкл")
	fs.StringVar(&p.LDAPURL, "ldap-url", "ldap://127.0.0.1:1389", "LDAP URL (backend ldap)")
	fs.StringVar(&p.LDAPBaseDN, "ldap-base-dn", "dc=oktopus,dc=dev", "LDAP base DN")
	fs.StringVar(&p.LDAPBindDN, "ldap-bind-dn", "cn=admin,dc=oktopus,dc=dev", "LDAP bind DN (service)")
	fs.StringVar(&p.LDAPBindPass, "ldap-bind-password", "admin_dev", "LDAP bind password")
}

func proxyConfigFromCLI(p proxyCLI) (proxy.Config, int) {
	mode := proxy.ConnectMode(p.Connect)
	switch mode {
	case proxy.ConnectTunnel, proxy.ConnectMITM:
	default:
		log.Printf("proxy: unknown -connect %q (use mitm or tunnel)", p.Connect)
		return proxy.Config{}, 2
	}

	backend := strings.ToLower(strings.TrimSpace(p.AuthBackend))
	if p.AuthEnable {
		if backend == "static" && p.AuthFile == "" {
			log.Print("proxy: auth backend static requires -auth-file")
			return proxy.Config{}, 2
		}
		if backend == "ldap" && p.LDAPURL == "" {
			log.Print("proxy: auth backend ldap requires -ldap-url")
			return proxy.Config{}, 2
		}
	}

	return proxy.Config{
		Listen:     p.Listen,
		Connect:    mode,
		CACertPath: p.CACert,
		CAKeyPath:  p.CAKey,
		Auth: proxy.AuthConfig{
			Enabled:    p.AuthEnable,
			Realm:      p.AuthRealm,
			Backend:    p.AuthBackend,
			StaticFile: p.AuthFile,
			CacheTTL:   p.AuthCacheTTL,
			LDAP: proxy.LDAPConfig{
				URL:          p.LDAPURL,
				BaseDN:       p.LDAPBaseDN,
				BindDN:       p.LDAPBindDN,
				BindPassword: p.LDAPBindPass,
			},
		},
	}, 0
}

func parseProxyFlags(name string, args []string) (proxyCLI, int) {
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	var p proxyCLI
	registerProxyFlags(fs, &p)
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return proxyCLI{}, 2
	}
	return p, 0
}
