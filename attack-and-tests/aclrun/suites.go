package aclrun

import "oktopus/attack-and-tests/setup"

// Домены с нормальным DNS (быстрый resolve, живые origin на 443 где нужен allow-probe).
const (
	domainBlocked  = "neverssl.com"
	domainAllow    = "example.com"
	domainAllowAlt = "iana.org"
	domainSuffix   = "www.example.com"
	domainWildApex = "wordpress.com"
	domainWildSub  = "www.wordpress.com"
	dstTestNet     = "203.0.113.9"
)

// RunAllItem — один тест в очереди run-all-acl.
type RunAllItem struct {
	Eval    *Suite
	Compile *CompileSuite
	Auth    *AuthSuite
	Inspect *InspectSuite
}

// AllSuites — порядок прогона run-all-acl.
func AllSuites() []RunAllItem {
	return []RunAllItem{
		{Eval: ptrSuite(HTTPAccessChain())},
		{Eval: ptrSuite(HTTPAccessNegation())},
		{Eval: ptrSuite(ACLAll())},
		{Eval: ptrSuite(ACLSrc())},
		{Eval: ptrSuite(ACLDst())},
		{Eval: ptrSuite(ACLDstDomain())},
		{Eval: ptrSuite(ACLPort())},
		{Eval: ptrSuite(ACLURLRegex())},
		{Eval: ptrSuite(ACLLdapGroup())},
		{Compile: ptrCompile(SSLVerify())},
		{Compile: ptrCompile(DelayAccess())},
		{Auth: ptrAuth(ProxyAuthStatic())},
		{Auth: ptrAuth(ProxyAuthLDAP())},
		{Inspect: ptrInspect(InspectAllowDeny())},
	}
}

func ptrInspect(v InspectSuite) *InspectSuite { return &v }

func ptrAuth(v AuthSuite) *AuthSuite { return &v }

func ptrSuite(v Suite) *Suite       { return &v }
func ptrCompile(v CompileSuite) *CompileSuite { return &v }

func HTTPAccessChain() Suite {
	return Suite{
		Name:   "http-access-chain",
		POCDir: POCDir("http-access-chain"),
		Cases: []Case{
			{Name: "deny listed host", Input: setup.EvaluateInput{SNI: domainBlocked, Path: "/", DstPort: 443}, WantAllowed: false},
			{Name: "allow other host", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/", DstPort: 443}, WantAllowed: true},
		},
	}
}

func HTTPAccessNegation() Suite {
	return Suite{
		Name:   "http-access-negation",
		POCDir: POCDir("http-access-negation"),
		Cases: []Case{
			{Name: "blocked host deny", Input: setup.EvaluateInput{SNI: domainBlocked, Path: "/"}, WantAllowed: false},
			{Name: "other host allow via !blocked", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/"}, WantAllowed: true},
		},
	}
}

func ACLAll() Suite {
	return Suite{
		Name:   "acl-all",
		POCDir: POCDir("acl-all"),
		Cases: []Case{
			{Name: "allow any", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/x", DstPort: 443}, WantAllowed: true},
		},
	}
}

func ACLSrc() Suite {
	return Suite{
		Name:   "acl-src",
		POCDir: POCDir("acl-src"),
		Cases: []Case{
			{Name: "deny 127.0.0.1", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/", SrcIP: "127.0.0.1"}, WantAllowed: false},
			{Name: "allow other 127.0.0.x", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/", SrcIP: "127.0.0.2"}, WantAllowed: true, SkipProxy: true},
			{Name: "unknown src deny", Input: setup.EvaluateInput{SNI: domainAllowAlt, Path: "/", SrcIP: "10.0.3.1"}, WantAllowed: false, SkipProxy: true},
		},
	}
}

func ACLDst() Suite {
	return Suite{
		Name:   "acl-dst",
		POCDir: POCDir("acl-dst"),
		Cases: []Case{
			{Name: "dst via sni literal deny", Input: setup.EvaluateInput{SNI: dstTestNet, Path: "/"}, WantAllowed: false},
			{Name: "ok domain allow", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/"}, WantAllowed: true},
		},
	}
}

func ACLDstDomain() Suite {
	return Suite{
		Name:   "acl-dstdomain",
		POCDir: POCDir("acl-dstdomain"),
		Cases: []Case{
			{Name: "exact deny", Input: setup.EvaluateInput{SNI: domainBlocked, Path: "/"}, WantAllowed: false},
			{Name: "suffix child deny", Input: setup.EvaluateInput{SNI: domainSuffix, Path: "/"}, WantAllowed: false},
			{Name: "wildcard apex not matched allow", Input: setup.EvaluateInput{SNI: domainWildApex, Path: "/"}, WantAllowed: true},
			{Name: "wildcard child deny", Input: setup.EvaluateInput{SNI: domainWildSub, Path: "/"}, WantAllowed: false},
			{Name: "unknown allow", Input: setup.EvaluateInput{SNI: domainAllowAlt, Path: "/"}, WantAllowed: true},
		},
	}
}

func ACLPort() Suite {
	return Suite{
		Name:   "acl-port",
		POCDir: POCDir("acl-port"),
		Cases: []Case{
			{Name: "deny port 22", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/", DstPort: 22}, WantAllowed: false},
			{Name: "allow other port", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/", DstPort: 2222}, WantAllowed: true},
			{Name: "deny in range", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/", DstPort: 8005}, WantAllowed: false},
			{Name: "allow below range", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/", DstPort: 7999}, WantAllowed: true},
		},
	}
}

func ACLURLRegex() Suite {
	const labPort = 9090
	host := "localhost"
	return Suite{
		Name:    "acl-url-regex",
		POCDir:  POCDir("acl-url-regex"),
		HTTPLab: true,
		Cases: []Case{
			{Name: "deny /secret path", Input: setup.EvaluateInput{SNI: host, Path: "/secret/data", DstPort: labPort}, WantAllowed: false, ProxyMode: setup.ProxyProbeHTTP},
			{Name: "allow /public", Input: setup.EvaluateInput{SNI: host, Path: "/public", DstPort: labPort}, WantAllowed: true, ProxyMode: setup.ProxyProbeHTTP},
		},
	}
}

func ACLLdapGroup() Suite {
	probe := setup.EvaluateInput{SNI: domainAllow, Path: "/", DstPort: 443}
	return Suite{
		Name:      "acl-ldap-group",
		POCDir:    POCDir("acl-ldap-group"),
		ProxyLDAP: true,
		Cases: []Case{
			{Name: "evaluate group allow_all", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/", Groups: []string{setup.LDAPDevGroupAllow}}, WantAllowed: true, SkipProxy: true},
			{Name: "evaluate group deny_all", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/", Groups: []string{setup.LDAPDevGroupDeny}}, WantAllowed: false, SkipProxy: true},
			{Name: "evaluate unknown group", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/", Groups: []string{"nogroup"}}, WantAllowed: false, SkipProxy: true},
			{Name: "ldap user in allow_all", Input: probe, WantAllowed: true, SkipEvaluate: true, ProxyUser: setup.LDAPDevUser, ProxyPass: setup.LDAPDevPass},
			{Name: "ldap user2 in deny_all", Input: probe, WantAllowed: false, SkipEvaluate: true, ProxyUser: setup.LDAPDevUser2, ProxyPass: setup.LDAPDevPass2},
		},
	}
}

func SSLVerify() CompileSuite {
	return CompileSuite{
		Name:   "ssl-verify",
		POCDir: POCDir("ssl-verify"),
		Smoke: []Case{
			{Name: "http_access still allows", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/"}, WantAllowed: true},
		},
	}
}

func DelayAccess() CompileSuite {
	return CompileSuite{
		Name:   "delay-access",
		POCDir: POCDir("delay-access"),
		Smoke: []Case{
			{Name: "http_access allow", Input: setup.EvaluateInput{SNI: domainAllow, Path: "/"}, WantAllowed: true},
		},
	}
}

func ProxyAuthStatic() AuthSuite {
	users := setup.ProxyAuthUser + ":" + setup.ProxyAuthPass
	return AuthSuite{
		Name:   "proxy-auth-static",
		POCDir: POCDir("proxy-auth-static"),
		Enable: func(client *setup.Client, connectMode string) (string, error) {
			return client.EnableProxyStaticAuth(connectMode, users)
		},
		Cases: []AuthCase{
			{Name: "listed user", User: setup.ProxyAuthUser, Pass: setup.ProxyAuthPass, WantAccept: true},
			{Name: "no credentials", User: "", Pass: "", WantAccept: false},
			{Name: "wrong password", User: setup.ProxyAuthUser, Pass: "wrong", WantAccept: false},
			{Name: "not in static list", User: "nobody-listed", Pass: "any", WantAccept: false},
		},
	}
}

const (
	inspectRuleDenySecretID  = "018f0000-0000-7000-8000-000000000101"
	inspectRuleAllowPublicID = "018f0000-0000-7000-8000-000000000102"
)

const inspectScriptDenySecret = `function inspect(ctx)
  return string.find(ctx.path, "/secret", 1, true) ~= nil
end`

const inspectScriptAllowPublic = `function inspect(ctx)
  return string.find(ctx.path, "/public", 1, true) ~= nil
end`

func InspectAllowDeny() InspectSuite {
	const labPort = 9090
	host := "localhost"
	return InspectSuite{
		Name:   "inspect-allow-deny",
		POCDir: POCDir("inspect-allow-deny"),
		Rules: []InspectRule{
			{
				ID: inspectRuleDenySecretID, Name: "deny-secret-path", Script: inspectScriptDenySecret,
				Action: 0, Enabled: true, SortOrder: 0,
			},
			{
				ID: inspectRuleAllowPublicID, Name: "allow-public-path", Script: inspectScriptAllowPublic,
				Action: 1, Enabled: true, SortOrder: 1,
			},
		},
		Cases: []Case{
			{
				Name:        "inspect deny /secret",
				Input:       setup.EvaluateInput{SNI: host, Path: "/secret/data", DstPort: labPort},
				WantAllowed: false,
				ProxyMode:   setup.ProxyProbeHTTP,
			},
			{
				Name:        "inspect allow /public (action allow on match)",
				Input:       setup.EvaluateInput{SNI: host, Path: "/public", DstPort: labPort},
				WantAllowed: true,
				ProxyMode:   setup.ProxyProbeHTTP,
			},
			{
				Name:        "inspect no match passes",
				Input:       setup.EvaluateInput{SNI: host, Path: "/other", DstPort: labPort},
				WantAllowed: true,
				ProxyMode:   setup.ProxyProbeHTTP,
			},
		},
	}
}

func ProxyAuthLDAP() AuthSuite {
	return AuthSuite{
		Name:        "proxy-auth-ldap",
		POCDir:      POCDir("proxy-auth-ldap"),
		RequireLDAP: true,
		Enable: func(client *setup.Client, connectMode string) (string, error) {
			return client.EnableProxyLDAPAuth(connectMode)
		},
		Cases: []AuthCase{
			{Name: "ldap user", User: setup.LDAPDevUser, Pass: setup.LDAPDevPass, WantAccept: true},
			{Name: "no credentials", User: "", Pass: "", WantAccept: false},
			{Name: "wrong password", User: setup.LDAPDevUser, Pass: "wrong", WantAccept: false},
			{Name: "unknown ldap user", User: "nobody-listed", Pass: "any", WantAccept: false},
		},
	}
}
