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
	if o.ConnectMode == "mitm" {
		log.Println("bypass: IPv6 loopback — ожидаем deny на CONNECT (не TLS lab)")
	}
	dest := "[::1]:9888"
	log.Printf("=== bypass: CONNECT %s ===", dest)
	conn, br, code, err := poclib.DialCONNECT(o.ProxyAddr, dest, o.ProxyUser, o.ProxyPass)
	if err != nil {
		log.Printf("CONNECT: HTTP %d %v", code, err)
		return false, nil
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	buf := make([]byte, 256)
	n, _ := io.ReadFull(br, buf)
	if n == 0 {
		_, _ = io.WriteString(conn, "\n")
		n2, _ := br.Read(buf)
		n = n2
	}
	if strings.Contains(string(buf[:n]), labMarker) {
		log.Println("bypass: ПОДТВЕРЖДЕНО")
		return true, nil
	}
	return false, nil
}
