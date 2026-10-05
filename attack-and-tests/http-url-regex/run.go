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
	ok, _ := runAttack(o)
	if ok {
		return 1
	}
	return 0
}

func runAttack(o poclib.RunOpts) (bool, error) {
	log.Println("=== control: /public (allow) ===")
	if !probePath(o, "/public", false) {
		log.Println("control: /public недоступен — проверьте lab")
	}
	log.Println("=== bypass: /secret/data (url_regex deny) ===")
	if probePath(o, "/secret/data", true) {
		return true, nil
	}
	return false, nil
}

func probePath(o poclib.RunOpts, path string, isBypass bool) bool {
	conn, err := net.Dial("tcp", o.ProxyAddr)
	if err != nil {
		return false
	}
	defer conn.Close()
	token := base64.StdEncoding.EncodeToString([]byte(o.ProxyUser + ":" + o.ProxyPass))
	raw := fmt.Sprintf(
		"GET http://localhost:9090%s HTTP/1.1\r\nHost: localhost\r\nProxy-Authorization: Basic %s\r\nConnection: close\r\n\r\n",
		path, token,
	)
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, _ = io.WriteString(conn, raw)
	buf, _ := io.ReadAll(conn)
	body := string(buf)
	log.Printf("GET %s: %s", path, setup.FormatPOCHTTPResponse(parseStatus(body), body))
	if isBypass {
		return strings.Contains(body, poclib.SecretMarker)
	}
	return strings.Contains(body, "PUBLIC_OK")
}

func parseStatus(body string) int {
	if i := strings.Index(body, " "); i > 0 && strings.HasPrefix(body, "HTTP/") {
		var code int
		fmt.Sscanf(body[i+1:], "%d", &code)
		return code
	}
	return 0
}
