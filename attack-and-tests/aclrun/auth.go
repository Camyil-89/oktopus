package aclrun

import (
	"log"

	"oktopus/attack-and-tests/setup"
)

// AuthCase — CONNECT с заданными учётными данными proxy-auth.
type AuthCase struct {
	Name       string
	User       string
	Pass       string
	WantAccept bool // true → HTTP 200, false → 407
}

// AuthSuite — проверка proxy-auth (static или ldap).
type AuthSuite struct {
	Name         string
	POCDir       string
	RequireLDAP  bool
	Enable       func(client *setup.Client, connectMode string) (listen string, err error)
	Cases        []AuthCase
}

// RunProxyAuthSuite публикует allow-all политику и гоняет кейсы auth. Exit: 0 OK, 1 FAIL.
func RunProxyAuthSuite(s AuthSuite, args []string) int {
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
	if s.RequireLDAP {
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
	log.Printf("%s: политика опубликована, %d auth-кейсов", s.Name, len(s.Cases))

	failed := false
	for _, mode := range proxyConnectModesAll {
		log.Printf("%s: connect=%s", s.Name, mode)
		proxyAddr, err := s.Enable(client, mode)
		if err != nil {
			log.Printf("%s: proxy setup (%s): %v", s.Name, mode, err)
			failed = true
			continue
		}
		log.Printf("%s: прокси %s (%s)", s.Name, proxyAddr, mode)
		for _, tc := range s.Cases {
			if runAuthCase(proxyAddr, tc) {
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

func runAuthCase(proxyAddr string, tc AuthCase) bool {
	status, err := setup.ProxyAuthCONNECTStatus(proxyAddr, tc.User, tc.Pass)
	if err != nil {
		log.Printf("  FAIL %s: connect: %v", tc.Name, err)
		return true
	}
	gotAccept := setup.ProxyAuthAccepted(status)
	if gotAccept != tc.WantAccept {
		if tc.WantAccept && status == 502 {
			log.Printf("  FAIL %s: status=%d (proxy auth backend error, LDAP?)", tc.Name, status)
		} else {
			log.Printf("  FAIL %s: status=%d wantAccept=%v", tc.Name, status, tc.WantAccept)
		}
		return true
	}
	if tc.WantAccept {
		log.Printf("  OK   %s auth accepted (CONNECT %d)", tc.Name, status)
	} else {
		log.Printf("  OK   %s auth denied (%d)", tc.Name, status)
	}
	return false
}
