package aclrun

import (
	"log"
	"path/filepath"

	"oktopus/attack-and-tests/setup"
)

// InspectRule — правило для sync через API.
type InspectRule struct {
	ID        string
	Name      string
	Script    string
	Action    int16 // 0 = deny on match, 1 = allow on match
	Enabled   bool
	SortOrder int
}

// InspectSuite — интеграция Lua-инспекции (только MITM).
type InspectSuite struct {
	Name   string
	POCDir string
	Rules  []InspectRule
	Cases  []Case
}

// RunInspectSuite публикует ACL, синхронизирует inspect rules, прогоняет HTTP-кейсы через прокси (mitm).
func RunInspectSuite(s InspectSuite, args []string) int {
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
		return 1
	}
	if err := client.PublishPolicy(policy); err != nil {
		log.Printf("%s: publish: %v", s.Name, err)
		return 1
	}
	syncInputs := inspectRulesToSync(s.Rules)
	if err := client.SyncInspectRules(syncInputs); err != nil {
		log.Printf("%s: inspect sync: %v", s.Name, err)
		return 1
	}

	log.Printf("%s: политика и %d правил инспекции опубликованы, %d кейсов", s.Name, len(s.Rules), len(s.Cases))

	stopHTTPLab, err := startACLHTTPLab()
	if err != nil {
		log.Printf("%s: http lab: %v", s.Name, err)
		return 1
	}
	defer stopHTTPLab()

	caPath := filepath.Join(s.POCDir, ".probe-ca.crt")
	if err := client.DownloadCACert(caPath); err != nil {
		log.Printf("%s: ca: %v", s.Name, err)
		return 1
	}
	setup.SetProbeCAPath(caPath)

	const mode = "mitm"
	log.Printf("%s: connect=%s (inspect только в MITM)", s.Name, mode)
	proxyAddr, err := client.EnableProxyACLTest(mode)
	if err != nil {
		log.Printf("%s: proxy setup: %v", s.Name, err)
		log.Println("RESULT: FAIL")
		return 1
	}
	log.Printf("%s: прокси %s (%s)", s.Name, proxyAddr, mode)

	failed := false
	for _, tc := range s.Cases {
		if tc.ProxyMode == "" {
			tc.ProxyMode = setup.ProxyProbeHTTP
		}
		if runProxyCase(proxyAddr, tc) {
			failed = true
		}
	}
	if failed {
		log.Println("RESULT: FAIL")
		return 1
	}
	log.Println("RESULT: OK")
	return 0
}

func inspectRulesToSync(rules []InspectRule) []setup.InspectRuleInput {
	out := make([]setup.InspectRuleInput, len(rules))
	for i, r := range rules {
		var idPtr *string
		if r.ID != "" {
			id := r.ID
			idPtr = &id
		}
		out[i] = setup.InspectRuleInput{
			ID:        idPtr,
			Name:      r.Name,
			Script:    r.Script,
			Action:    r.Action,
			Enabled:   r.Enabled,
			SortOrder: r.SortOrder,
		}
	}
	return out
}
