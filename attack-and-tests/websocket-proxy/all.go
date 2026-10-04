package main

import (
	"log"
	"path/filepath"

	"oktopus/attack-and-tests/setup"
)

const pocName = "websocket-proxy"

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
	log.Println("lab: wss :9443 и ws :9080 в фоне")

	pocDir := setup.POCDirFromCaller(pocName)
	allOK := true
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
			if mode == "mitm" {
				caPath := filepath.Join(pocDir, ".ca.crt")
				if err := client.DownloadCACert(caPath); err != nil {
					log.Printf("ca: %v", err)
					exitSetupErr = true
					break
				}
				run.caFile = caPath
			}
			log.Printf("setup: proxy=%s connect=%s auth=%s:***", listen, mode, setup.ProxyAuthUser)
		}

		wssOK, wsOK := runProbes(run)
		if wssOK && wsOK {
			log.Printf("connect_mode=%s: OK", mode)
		} else {
			allOK = false
			log.Printf("connect_mode=%s: FAIL", mode)
		}
	}

	log.Println("")
	if exitSetupErr {
		log.Println("RESULT: FAIL (ошибка настройки)")
		return 1
	}
	if allOK {
		log.Println("RESULT: OK")
		return 0
	}
	log.Println("RESULT: FAIL")
	return 1
}
