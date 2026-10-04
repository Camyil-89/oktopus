package main

import (
	"log"
)

func cmdRun(args []string) int {
	log.SetFlags(0)
	o, err := parseRunFlags(args)
	if err != nil {
		return 2
	}
	if o.connectMode == "" {
		o.connectMode = "mitm"
	}
	okWSS, okWS := runProbes(o)
	if okWSS && okWS {
		return 0
	}
	return 1
}

func runProbes(o runOpts) (wssOK, wsOK bool) {
	log.Printf("=== wss (CONNECT + %s + WebSocket) ===", o.connectMode)
	if err := probeWSSConnect(o); err != nil {
		log.Printf("wss: FAIL — %v", err)
	} else {
		log.Println("wss: OK")
		wssOK = true
	}

	log.Println("=== ws (plain HTTP proxy, absolute URL) ===")
	if err := probeWSPlainHTTPProxy(o); err != nil {
		log.Printf("ws: FAIL — %v", err)
	} else {
		log.Println("ws: OK")
		wsOK = true
	}
	return wssOK, wsOK
}
