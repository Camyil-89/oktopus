package main

import (
	"log"
	"net/http"

	"oktopus/attack-and-tests/poclib"
)

func cmdRun(args []string) int {
	log.SetFlags(0)
	o, err := parseFlags(args)
	if err != nil {
		return 2
	}
	ok, _ := runAttack(o)
	if ok {
		return 1
	}
	return 0
}

func runAttack(o poclib.RunOpts) (bool, error) {
	log.Println("=== control: CONNECT с Proxy-Authorization (ожидаем 200) ===")
	_, _, code, err := poclib.DialCONNECT(o.ProxyAddr, "localhost:9443", o.ProxyUser, o.ProxyPass)
	if err != nil || code != http.StatusOK {
		log.Printf("control: с auth CONNECT не 200 (%d %v)", code, err)
	}

	log.Println("=== bypass: CONNECT без Proxy-Authorization ===")
	code, err = poclib.DialCONNECTNoAuth(o.ProxyAddr, "localhost:9443")
	if err != nil {
		log.Printf("CONNECT: %v", err)
	}
	switch code {
	case http.StatusProxyAuthRequired:
		log.Println("bypass: 407 — ожидаемо")
		return false, nil
	case http.StatusOK:
		log.Println("bypass: ПОДТВЕРЖДЕНО — туннель без auth")
		return true, nil
	default:
		log.Printf("bypass: HTTP %d", code)
		return code == http.StatusOK, nil
	}
}
