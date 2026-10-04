package aclrun

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"oktopus/attack-and-tests/setup"
)

// Case — evaluate + (опционально) запрос через прокси.
type Case struct {
	Name        string
	Input       setup.EvaluateInput
	WantAllowed bool
	SkipProxy    bool // без запроса через прокси
	SkipEvaluate bool // без /api/proxy/acl/evaluate (реальный LDAP → группы на прокси)
	ProxyMode   setup.ProxyProbeMode
	ProxyUser   string // Proxy-Authorization (пусто → attack-poc)
	ProxyPass   string
}

// Suite — интеграционный тест http_access.
type Suite struct {
	Name        string
	POCDir      string
	ProxyLDAP   bool // прокси с auth backend ldap (нужен dev OpenLDAP)
	HTTPLab     bool // plain HTTP lab на 127.0.0.1:9090 для url_regex proxy probe
	Cases       []Case
}

// CompileSuite — validate + publish + smoke.
type CompileSuite struct {
	Name   string
	POCDir string
	Smoke  []Case
}

// ParseFlags общие флаги API.
func ParseFlags(args []string) (setup.APIFlags, []string, error) {
	fs := flag.NewFlagSet("acl-policy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	api := setup.DefaultAPIFlags()
	fs.StringVar(&api.APIBase, "api", api.APIBase, "базовый URL API")
	fs.StringVar(&api.APIUser, "api-user", api.APIUser, "логин UI")
	fs.StringVar(&api.APIPass, "api-pass", api.APIPass, "пароль UI")
	if err := fs.Parse(args); err != nil {
		return setup.APIFlags{}, nil, err
	}
	return api, fs.Args(), nil
}

var proxyConnectModesAll = []string{"mitm", "tunnel"}

// RunEvaluateSuite публикует policy и прогоняет кейсы. Exit: 0 OK, 1 FAIL.
func RunEvaluateSuite(s Suite, args []string) int {
	log.SetFlags(0)
	api, _, err := ParseFlags(args)
	if err != nil {
		return 2
	}
	policy, err := setup.LoadPolicyFile(s.POCDir)
	if err != nil {
		log.Printf("%s: policy: %v", s.Name, err)
		return 1
	}
	client, err := setup.NewClient(api.APIBase)
	if err != nil {
		log.Printf("%s: api: %v", s.Name, err)
		return 1
	}
	if err := client.Login(api.APIUser, api.APIPass); err != nil {
		log.Printf("%s: login: %v", s.Name, err)
		return 1
	}
	if _, err := client.EnsureTestInstance(); err != nil {
		log.Printf("%s: test instance: %v", s.Name, err)
		return 1
	}
	if s.ProxyLDAP {
		if err := setup.CheckLDAPDevReachable(); err != nil {
			log.Printf("%s: %v", s.Name, err)
			return 1
		}
	}
	if failed := validatePolicy(client, s.Name, policy); failed {
		return 1
	}
	if err := client.PublishPolicy(policy); err != nil {
		log.Printf("%s: publish: %v", s.Name, err)
		return 1
	}
	log.Printf("%s: политика опубликована, %d кейсов", s.Name, len(s.Cases))

	var stopHTTPLab func()
	if s.HTTPLab {
		stopHTTPLab, err = startACLHTTPLab()
		if err != nil {
			log.Printf("%s: http lab: %v", s.Name, err)
			return 1
		}
		defer stopHTTPLab()
	}

	caPath := filepath.Join(s.POCDir, ".probe-ca.crt")
	if err := client.DownloadCACert(caPath); err != nil {
		log.Printf("%s: ca: %v", s.Name, err)
		return 1
	}
	setup.SetProbeCAPath(caPath)

	failed := false
	for _, tc := range s.Cases {
		if tc.SkipEvaluate {
			continue
		}
		if runEvaluateCase(client, tc) {
			failed = true
		}
	}
	for _, mode := range proxyConnectModesAll {
		log.Printf("%s: connect=%s", s.Name, mode)
		proxyAddr, err := enableProxyForSuite(client, s, mode)
		if err != nil {
			log.Printf("%s: proxy setup (%s): %v", s.Name, mode, err)
			failed = true
			continue
		}
		log.Printf("%s: прокси %s (%s)", s.Name, proxyAddr, mode)
		for _, tc := range s.Cases {
			if runProxyCase(proxyAddr, tc) {
				failed = true
			}
		}
	}
	if failed {
		log.Println("RESULT: FAIL")
		return 1
	}
	log.Println("RESULT: OK")
	return 0
}

func enableProxyForSuite(client *setup.Client, s Suite, connectMode string) (string, error) {
	if s.ProxyLDAP {
		return client.EnableProxyLDAPAuth(connectMode)
	}
	return client.EnableProxyACLTest(connectMode)
}

func RunCompileSuite(s CompileSuite, args []string) int {
	log.SetFlags(0)
	api, _, err := ParseFlags(args)
	if err != nil {
		return 2
	}
	policy, err := setup.LoadPolicyFile(s.POCDir)
	if err != nil {
		log.Printf("%s: policy: %v", s.Name, err)
		return 1
	}
	client, err := setup.NewClient(api.APIBase)
	if err != nil {
		log.Printf("%s: api: %v", s.Name, err)
		return 1
	}
	if err := client.Login(api.APIUser, api.APIPass); err != nil {
		log.Printf("%s: login: %v", s.Name, err)
		return 1
	}
	if _, err := client.EnsureTestInstance(); err != nil {
		log.Printf("%s: test instance: %v", s.Name, err)
		return 1
	}
	if failed := validatePolicy(client, s.Name, policy); failed {
		log.Println("RESULT: FAIL")
		return 1
	}
	log.Printf("%s: validate OK", s.Name)
	if err := client.PublishPolicy(policy); err != nil {
		log.Printf("%s: publish: %v", s.Name, err)
		log.Println("RESULT: FAIL")
		return 1
	}
	log.Printf("%s: publish OK", s.Name)

	if len(s.Smoke) == 0 {
		log.Println("RESULT: OK")
		return 0
	}
	caPath := filepath.Join(s.POCDir, ".probe-ca.crt")
	if err := client.DownloadCACert(caPath); err != nil {
		log.Printf("%s: ca: %v", s.Name, err)
		log.Println("RESULT: FAIL")
		return 1
	}
	setup.SetProbeCAPath(caPath)

	failed := false
	for _, tc := range s.Smoke {
		if runEvaluateCase(client, tc) {
			failed = true
		}
	}
	for _, mode := range proxyConnectModesAll {
		log.Printf("%s: connect=%s", s.Name, mode)
		proxyAddr, err := client.EnableProxyACLTest(mode)
		if err != nil {
			log.Printf("%s: proxy setup (%s): %v", s.Name, mode, err)
			failed = true
			continue
		}
		for _, tc := range s.Smoke {
			if runProxyCase(proxyAddr, tc) {
				failed = true
			}
		}
	}
	if failed {
		log.Println("RESULT: FAIL")
		return 1
	}
	log.Println("RESULT: OK")
	return 0
}

func validatePolicy(client *setup.Client, name, policy string) bool {
	an, err := client.ValidatePolicy(policy)
	if err != nil {
		log.Printf("%s: validate: %v", name, err)
		return true
	}
	if !an.OK {
		log.Printf("%s: validate: политика не ok", name)
		for _, d := range an.Diagnostics {
			if d.Severity == "error" {
				log.Printf("  L%d: %s", d.Line, d.Message)
			}
		}
		return true
	}
	return false
}

func runEvaluateCase(client *setup.Client, tc Case) bool {
	res, err := client.Evaluate(tc.Input)
	if err != nil {
		log.Printf("  FAIL %s: evaluate: %v", tc.Name, err)
		return true
	}
	if res.Allowed != tc.WantAllowed {
		log.Printf("  FAIL %s: evaluate allowed=%v want=%v", tc.Name, res.Allowed, tc.WantAllowed)
		return true
	}
	log.Printf("  OK   %s evaluate (allowed=%v)", tc.Name, res.Allowed)
	return false
}

func runProxyCase(proxyAddr string, tc Case) bool {
	if tc.SkipProxy {
		return false
	}
	got, err := setup.ProxyProbeAllowed(proxyAddr, tc.Input, tc.ProxyMode, tc.ProxyUser, tc.ProxyPass)
	if err != nil {
		if !tc.WantAllowed && setup.ConnectErrorMeansDenied(err) {
			log.Printf("  OK   %s proxy (deny: %v)", tc.Name, err)
			return false
		}
		log.Printf("  FAIL %s: proxy: %v", tc.Name, err)
		return true
	}
	if got != tc.WantAllowed {
		log.Printf("  FAIL %s: proxy allowed=%v want=%v", tc.Name, got, tc.WantAllowed)
		return true
	}
	log.Printf("  OK   %s proxy (allowed=%v)", tc.Name, got)
	return false
}

// POCDir путь от корня репозитория.
func POCDir(name string) string {
	return fmt.Sprintf("attack-and-tests/%s", name)
}
