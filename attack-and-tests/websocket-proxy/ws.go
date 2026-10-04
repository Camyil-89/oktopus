package main

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"net/url"
	"os"
	"strings"
)

func tlsConfigForWSS(o runOpts) (*tls.Config, error) {
	cfg := &tls.Config{
		ServerName: o.wssSNI,
		MinVersion: tls.VersionTLS12,
	}
	switch o.connectMode {
	case "tunnel":
		// Прозрачный tunnel: TLS с origin lab (самоподписанный), как в mitm-host-mismatch.
		cfg.InsecureSkipVerify = true
		return cfg, nil
	case "mitm", "":
		pool, err := loadCA(o.caFile)
		if err != nil {
			return nil, fmt.Errorf("ca: %w", err)
		}
		cfg.RootCAs = pool
		return cfg, nil
	default:
		return nil, fmt.Errorf("connect-mode: %q", o.connectMode)
	}
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

// probeWSSConnect: CONNECT → TLS → WebSocket upgrade → echo (MITM: CA прокси; tunnel: cert lab origin).
func probeWSSConnect(o runOpts) error {
	tlsCfg, err := tlsConfigForWSS(o)
	if err != nil {
		return err
	}

	raw, br, err := dialCONNECT(o.proxyAddr, o.wssConnect, o.proxyUser, o.proxyPass)
	if err != nil {
		return err
	}
	defer raw.Close()

	tlsConn := tls.Client(&bufConn{Conn: raw, r: br}, tlsCfg)
	if err := tlsConn.Handshake(); err != nil {
		return fmt.Errorf("TLS: %w", err)
	}

	host := o.wssConnect
	if !strings.Contains(host, ":") {
		host += ":443"
	}
	if err := writeWebSocketUpgradeRequest(tlsConn, host, "https://"+host, "/"); err != nil {
		return err
	}
	r := bufio.NewReader(tlsConn)
	if err := expectSwitchingProtocols(r); err != nil {
		return err
	}
	if err := writeClientTextFrame(tlsConn, "wss-ping"); err != nil {
		return err
	}
	got, err := readServerTextFrame(r)
	if err != nil {
		return err
	}
	if got != "echo:wss-ping" {
		return fmt.Errorf("wss echo: got %q", got)
	}
	return nil
}

// probeWSPlainHTTPProxy: GET absolute URL на прокси с Upgrade (explicit proxy).
func probeWSPlainHTTPProxy(o runOpts) error {
	u, err := url.Parse(o.wsHTTPURL)
	if err != nil {
		return err
	}
	host := u.Host
	if host == "" {
		return fmt.Errorf("ws-url: missing host")
	}
	path := u.Path
	if path == "" {
		path = "/"
	}

	conn, err := net.Dial("tcp", o.proxyAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	origin := u.Scheme + "://" + host
	if err := writeWebSocketUpgradeRequestToProxy(conn, o.wsHTTPURL, host, origin, path, o.proxyUser, o.proxyPass); err != nil {
		return err
	}
	br := bufio.NewReader(conn)
	if err := expectSwitchingProtocols(br); err != nil {
		return err
	}
	if err := writeClientTextFrame(conn, "ws-ping"); err != nil {
		return err
	}
	got, err := readServerTextFrame(br)
	if err != nil {
		return err
	}
	if got != "echo:ws-ping" {
		return fmt.Errorf("ws echo: got %q", got)
	}
	return nil
}

func writeWebSocketUpgradeRequest(w io.Writer, host, origin, path string) error {
	key, err := randomSecWebSocketKey()
	if err != nil {
		return err
	}
	req := fmt.Sprintf(
		"GET %s HTTP/1.1\r\nHost: %s\r\nOrigin: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		path, host, origin, key,
	)
	_, err = io.WriteString(w, req)
	return err
}

func writeWebSocketUpgradeRequestToProxy(w io.Writer, absoluteURL, host, origin, path, user, pass string) error {
	key, err := randomSecWebSocketKey()
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("GET %s HTTP/1.1\r\n", absoluteURL))
	b.WriteString(fmt.Sprintf("Host: %s\r\n", host))
	if user != "" || pass != "" {
		token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		b.WriteString("Proxy-Authorization: Basic " + token + "\r\n")
	}
	b.WriteString(fmt.Sprintf("Origin: %s\r\n", origin))
	b.WriteString("Upgrade: websocket\r\n")
	b.WriteString("Connection: Upgrade\r\n")
	b.WriteString("Sec-WebSocket-Key: " + key + "\r\n")
	b.WriteString("Sec-WebSocket-Version: 13\r\n")
	b.WriteString("\r\n")
	_, err = io.WriteString(w, b.String())
	return err
}

func randomSecWebSocketKey() (string, error) {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

func expectSwitchingProtocols(br *bufio.Reader) error {
	res, err := stdhttp.ReadResponse(br, nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != stdhttp.StatusSwitchingProtocols {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("upgrade: HTTP %d body %q", res.StatusCode, body)
	}
	return nil
}

func writeClientTextFrame(w io.Writer, payload string) error {
	p := []byte(payload)
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	masked := make([]byte, len(p))
	for i := range p {
		masked[i] = p[i] ^ mask[i%4]
	}
	hdr := []byte{0x81, byte(0x80 | len(p))}
	if _, err := w.Write(hdr); err != nil {
		return err
	}
	if _, err := w.Write(mask); err != nil {
		return err
	}
	_, err := w.Write(masked)
	return err
}

func readServerTextFrame(r *bufio.Reader) (string, error) {
	h, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	if h&0x0f != 0x01 {
		return "", fmt.Errorf("opcode %d", h&0x0f)
	}
	lenByte, err := r.ReadByte()
	if err != nil {
		return "", err
	}
	n := int(lenByte & 0x7f)
	if n == 126 {
		var ext [2]byte
		if _, err := io.ReadFull(r, ext[:]); err != nil {
			return "", err
		}
		n = int(ext[0])<<8 | int(ext[1])
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}
