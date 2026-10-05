package main

import (
	"fmt"
	"log"

	"oktopus/attack-and-tests/poclib"
)

func cmdRun(args []string) int {
	log.SetFlags(0)
	o, err := parseFlags(args)
	if err != nil {
		return 2
	}
	ok, err := runAttack(o)
	if err != nil {
		log.Printf("run: %v", err)
		return 1
	}
	if ok {
		return 1
	}
	return 0
}

func runAttack(o poclib.RunOpts) (bool, error) {
	if o.ConnectMode == "tunnel" {
		log.Println("bypass: port defer — сценарий для MITM (defer PORT на CONNECT)")
		return false, nil
	}
	pool, err := poclib.LoadCAForMode(o.CAFile, o.ConnectMode)
	if err != nil {
		return false, err
	}

	log.Println("=== bypass: CONNECT localhost:9443, GET https://localhost:9778/secret ===")
	raw, br, _, err := poclib.DialCONNECT(o.ProxyAddr, "localhost:9443", o.ProxyUser, o.ProxyPass)
	if err != nil {
		log.Printf("CONNECT: %v", err)
		return false, nil
	}
	defer raw.Close()

	tlsConn, err := poclib.TLSOverCONNECT(raw, br, o.ConnectMode, "localhost", pool)
	if err != nil {
		log.Printf("TLS: %v", err)
		return false, nil
	}
	req := fmt.Sprintf("GET https://127.0.0.1:9778/secret HTTP/1.1\r\nHost: 127.0.0.1:9778\r\nConnection: close\r\n\r\n")
	st, body, err := poclib.SendRawHTTPAndRead(tlsConn, req)
	if err != nil {
		log.Printf("HTTP: %v", err)
		return false, nil
	}
	poclib.LogBypassResponse(st, body)
	if poclib.BypassVerdict(st, body, "127.0.0.1:9778") {
		return true, nil
	}
	return false, nil
}
