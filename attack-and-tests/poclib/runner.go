package poclib

import (
	"log"
	"path/filepath"

	"oktopus/attack-and-tests/setup"
)

// RunOpts — общие флаги PoC (API, прокси, режим).
type RunOpts struct {
	ProxyAddr   string
	ProxyUser   string
	ProxyPass   string
	CAFile      string
	ConnectMode string
	API         setup.APIFlags
	SkipSetup   bool
}

// RunDualMode поднимает lab снаружи, для mitm и tunnel настраивает прокси и вызывает attack.
// attack возвращает true, если обход удался (уязвимость).
func RunDualMode(pocName string, o RunOpts, attack func(RunOpts) (bool, error)) int {
	pocDir := setup.POCDirFromCaller(pocName)
	anyAttackOK := false
	exitSetupErr := false

	for _, mode := range []string{"mitm", "tunnel"} {
		log.Println("")
		log.Printf("======== connect_mode=%s ========", mode)
		run := o
		run.ConnectMode = mode
		if !run.SkipSetup {
			client, err := setup.NewClient(run.API.APIBase)
			if err != nil {
				log.Printf("api: %v", err)
				exitSetupErr = true
				break
			}
			if err := client.Login(run.API.APIUser, run.API.APIPass); err != nil {
				log.Printf("api login: %v", err)
				exitSetupErr = true
				break
			}
			acl, err := setup.LoadACLExample(pocDir)
			if err != nil {
				log.Printf("acl: %v", err)
				exitSetupErr = true
				break
			}
			listen, err := client.ApplyProxyTestProfile(mode, acl)
			if err != nil {
				log.Printf("setup %s: %v", mode, err)
				exitSetupErr = true
				break
			}
			run.ProxyAddr = listen
			run.ProxyUser = setup.ProxyAuthUser
			run.ProxyPass = setup.ProxyAuthPass
			caPath := filepath.Join(pocDir, ".ca.crt")
			if mode == "mitm" {
				if err := client.DownloadCACert(caPath); err != nil {
					log.Printf("ca: %v", err)
					exitSetupErr = true
					break
				}
				run.CAFile = caPath
			}
			log.Printf("setup: proxy=%s auth=%s:***", listen, setup.ProxyAuthUser)
		}

		succeeded, err := attack(run)
		if err != nil {
			log.Printf("run: %v", err)
			exitSetupErr = true
			break
		}
		if succeeded {
			anyAttackOK = true
			log.Printf("connect_mode=%s: FAIL (атака удалась)", mode)
		} else {
			log.Printf("connect_mode=%s: OK (атака не удалась)", mode)
		}
	}

	log.Println("")
	if exitSetupErr {
		log.Println("RESULT: FAIL (ошибка настройки или прогона)")
		return 1
	}
	if anyAttackOK {
		log.Println("RESULT: FAIL")
		return 1
	}
	log.Println("RESULT: OK")
	return 0
}

// ParseAPIFlags — -api, -api-user, -api-pass, -skip-setup.
func ParseAPIFlags(fs interface{ StringVar(*string, string, string, string); BoolVar(*bool, string, bool, string) }, api *setup.APIFlags, skipSetup *bool) {
	def := setup.DefaultAPIFlags()
	*api = def
	fs.StringVar(&api.APIBase, "api", api.APIBase, "базовый URL API")
	fs.StringVar(&api.APIUser, "api-user", api.APIUser, "логин UI")
	fs.StringVar(&api.APIPass, "api-pass", api.APIPass, "пароль UI")
	fs.BoolVar(skipSetup, "skip-setup", false, "не трогать API")
}
