package main

import (
	"log"
	"path/filepath"

	"oktopus/attack-and-tests/setup"
)

const pocName = "mitm-allowed-sni-evil-host"

func cmdAll(args []string) int {
	log.SetFlags(0)
	o, err := parseRunFlags(args)
	if err != nil {
		return 2
	}

	stopLab, err := startLabServers()
	if err != nil {
		log.Printf("lab: %v", err)
		return 1
	}
	defer stopLab()
	log.Println("lab: origin :9443 и :9555 в фоне")

	pocDir := setup.POCDirFromCaller(pocName)
	anyAttackOK := false
	exitSetupErr := false

	for _, mode := range []string{"mitm", "tunnel"} {
		log.Println("")
		log.Printf("======== connect_mode=%s ========", mode)
		run := o
		run.connectMode = mode
		if !run.skipSetup {
			client, err := setup.NewClient(run.api.APIBase)
			if err != nil {
				log.Printf("api: %v", err)
				exitSetupErr = true
				break
			}
			if err := client.Login(run.api.APIUser, run.api.APIPass); err != nil {
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
			run.proxyAddr = listen
			run.proxyUser = setup.ProxyAuthUser
			run.proxyPass = setup.ProxyAuthPass
			caPath := filepath.Join(pocDir, ".ca.crt")
			if mode == "mitm" {
				if err := client.DownloadCACert(caPath); err != nil {
					log.Printf("ca: %v", err)
					exitSetupErr = true
					break
				}
				run.caFile = caPath
			}
			log.Printf("setup: proxy=%s auth=%s:***", listen, setup.ProxyAuthUser)
		}

		succeeded, err := runAttack(run)
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
