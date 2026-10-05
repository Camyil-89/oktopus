package poclib

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
	"time"

	"oktopus/attack-and-tests/setup"
)

const SecretMarker = "SECRET_INTERNAL_HIT"

// LoadCAForMode — CA для mitm; tunnel — пустой pool.
func LoadCAForMode(path, connectMode string) (*x509.CertPool, error) {
	if connectMode == "tunnel" {
		return x509.NewCertPool(), nil
	}
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

// TLSOverCONNECT поднимает TLS поверх CONNECT (MITM CA или skip verify в tunnel).
func TLSOverCONNECT(raw net.Conn, br *bufio.Reader, connectMode, sni string, pool *x509.CertPool) (*tls.Conn, error) {
	tlsCfg := &tls.Config{
		ServerName: sni,
		MinVersion: tls.VersionTLS12,
	}
	if connectMode == "tunnel" {
		tlsCfg.InsecureSkipVerify = true
	} else {
		tlsCfg.RootCAs = pool
	}
	tlsConn := tls.Client(&BufConn{Conn: raw, R: br}, tlsCfg)
	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}
	return tlsConn, nil
}

// SendRawHTTPAndRead отправляет сырой HTTP/1.1 и читает один ответ.
func SendRawHTTPAndRead(conn net.Conn, rawReq string) (status int, body string, err error) {
	if _, err := conn.Write([]byte(rawReq)); err != nil {
		return 0, "", err
	}
	return ReadHTTPResponse(conn)
}

// ReadHTTPResponse читает один HTTP/1.1 ответ.
func ReadHTTPResponse(conn net.Conn) (status int, body string, err error) {
	br := bufio.NewReader(conn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		return 0, "", err
	}
	parts := strings.SplitN(strings.TrimSpace(statusLine), " ", 3)
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("bad status: %q", statusLine)
	}
	code := 0
	fmt.Sscanf(parts[1], "%d", &code)

	var contentLen int
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return 0, "", err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			fmt.Sscanf(line, "Content-Length: %d", &contentLen)
		}
	}
	var bodyBytes []byte
	if contentLen > 0 {
		bodyBytes = make([]byte, contentLen)
		_, err = io.ReadFull(br, bodyBytes)
	} else {
		bodyBytes, err = io.ReadAll(br)
	}
	if err != nil && err != io.EOF {
		return 0, "", err
	}
	return code, string(bodyBytes), nil
}

// BypassVerdict оценивает ответ bypass-атаки на internal.
func BypassVerdict(status int, body, evilHost string) bool {
	if strings.Contains(body, SecretMarker) {
		return true
	}
	if evilHost != "" && strings.Contains(body, evilHost) {
		if status == 502 || strings.Contains(body, "dial tcp") || strings.Contains(body, "connectex") {
			return true
		}
	}
	return false
}

// LogBypassResponse логирует ответ bypass.
func LogBypassResponse(status int, body string) {
	if snippet := setup.POCResponseSnippet(status, body); snippet != "" {
		log.Printf("bypass: HTTP %d — %s", status, snippet)
	} else {
		log.Printf("bypass: HTTP %d", status)
	}
}

// BufConn — читает остаток буфера после CONNECT.
type BufConn struct {
	Conn net.Conn
	R    *bufio.Reader
}

func (b *BufConn) Read(p []byte) (int, error) {
	return b.R.Read(p)
}

func (b *BufConn) Write(p []byte) (int, error) {
	return b.Conn.Write(p)
}

func (b *BufConn) Close() error {
	return b.Conn.Close()
}

func (b *BufConn) LocalAddr() net.Addr                { return b.Conn.LocalAddr() }
func (b *BufConn) RemoteAddr() net.Addr               { return b.Conn.RemoteAddr() }
func (b *BufConn) SetDeadline(t time.Time) error      { return b.Conn.SetDeadline(t) }
func (b *BufConn) SetReadDeadline(t time.Time) error  { return b.Conn.SetReadDeadline(t) }
func (b *BufConn) SetWriteDeadline(t time.Time) error { return b.Conn.SetWriteDeadline(t) }
