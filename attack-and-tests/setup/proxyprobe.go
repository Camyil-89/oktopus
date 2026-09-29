package setup

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var probeCAPath string

// SetProbeCAPath задаёт CA MITM прокси для проверки Forbidden после CONNECT 200.
func SetProbeCAPath(path string) {
	probeCAPath = strings.TrimSpace(path)
}

// Таймауты интеграционных ACL-probe (не PoC атак).
const proxyProbeTimeout = 5 * time.Second

const connectResponseTimeout = 4 * time.Second

const proxyTLSProbeTimeout = 2 * time.Second

// EnableProxyACLTest включает прокси и auth (политика уже опубликована).
func (c *Client) EnableProxyACLTest(connectMode string) (listen string, err error) {
	users := ProxyAuthUser + ":" + ProxyAuthPass
	return c.EnableProxyStaticAuth(connectMode, users)
}

// ProxyProbeMode — как проверять кейс через прокси.
type ProxyProbeMode string

const (
	ProxyProbeAuto    ProxyProbeMode = "auto"
	ProxyProbeCONNECT ProxyProbeMode = "connect"
	ProxyProbeHTTP    ProxyProbeMode = "http"
)

func probeProxyAuth(user, pass string) (string, string) {
	if user != "" || pass != "" {
		return user, pass
	}
	return ProxyAuthUser, ProxyAuthPass
}

// ProxyProbeAllowed выполняет запрос через прокси; true = ACL пропустил (как evaluate allowed).
// proxyUser/proxyPass — Proxy-Authorization; пустые → attack-poc (static ACL-тесты).
func ProxyProbeAllowed(proxyAddr string, in EvaluateInput, mode ProxyProbeMode, proxyUser, proxyPass string) (bool, error) {
	proxyUser, proxyPass = probeProxyAuth(proxyUser, proxyPass)
	if mode == ProxyProbeAuto {
		path := strings.TrimSpace(in.Path)
		if path != "" && path != "/" {
			mode = ProxyProbeHTTP
		} else {
			mode = ProxyProbeCONNECT
		}
	}
	switch mode {
	case ProxyProbeHTTP:
		return proxyHTTPAllowed(proxyAddr, in, proxyUser, proxyPass)
	default:
		return proxyCONNECTAllowed(proxyAddr, in, proxyUser, proxyPass)
	}
}

func proxyCONNECTAllowed(proxyAddr string, in EvaluateInput, proxyUser, proxyPass string) (bool, error) {
	host := strings.TrimSpace(in.SNI)
	if host == "" {
		host = "localhost"
	}
	port := in.DstPort
	if port == 0 {
		port = 443
	}
	dest := fmt.Sprintf("%s:%d", host, port)
	status, conn, err := dialCONNECTProbe(proxyAddr, dest, proxyUser, proxyPass)
	if conn != nil {
		defer conn.Close()
	}
	if status == http.StatusProxyAuthRequired {
		return false, fmt.Errorf("proxy auth required (407)")
	}
	if status == http.StatusForbidden {
		return false, nil
	}
	if err != nil {
		if status == http.StatusForbidden || strings.Contains(err.Error(), "403") {
			return false, nil
		}
		return false, fmt.Errorf("CONNECT %s: %w", dest, err)
	}
	if status != http.StatusOK || conn == nil {
		return false, fmt.Errorf("CONNECT %s: unexpected HTTP %d", dest, status)
	}
	if probeCAPath == "" {
		return true, nil
	}
	denied, perr := connectTunnelLooksDenied(conn, host, port)
	if perr != nil {
		return true, nil
	}
	if denied {
		return false, nil
	}
	return true, nil
}

func connectHTTPHost(host string, port int) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "localhost"
	}
	if port == 0 || port == 443 {
		return host
	}
	if strings.Contains(host, ":") {
		return host
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", port))
}

func connectTunnelLooksDenied(raw net.Conn, sni string, dstPort int) (denied bool, err error) {
	pool, err := loadProbeCA()
	if err != nil {
		return false, err
	}
	deadline := time.Now().Add(proxyTLSProbeTimeout)
	_ = raw.SetDeadline(deadline)
	tlsConn := tls.Client(raw, &tls.Config{
		RootCAs:    pool,
		ServerName: sni,
		MinVersion: tls.VersionTLS12,
	})
	hctx, cancel := context.WithTimeout(context.Background(), proxyTLSProbeTimeout)
	defer cancel()
	if err := tlsConn.HandshakeContext(hctx); err != nil {
		return false, nil
	}
	_ = tlsConn.SetDeadline(deadline)
	hostHeader := connectHTTPHost(sni, dstPort)
	req := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", hostHeader)
	if _, err := tlsConn.Write([]byte(req)); err != nil {
		return false, err
	}
	br := bufio.NewReader(tlsConn)
	statusLine, err := br.ReadString('\n')
	if err != nil {
		return false, err
	}
	code := parseHTTPStatus(statusLine)
	body, _ := io.ReadAll(io.LimitReader(br, 4096))
	if looksForbiddenHTTP(code, string(body)) {
		return true, nil
	}
	return false, nil
}

func loadProbeCA() (*x509.CertPool, error) {
	pem, err := os.ReadFile(probeCAPath)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("no certs in %s", probeCAPath)
	}
	return pool, nil
}

func parseHTTPStatus(statusLine string) int {
	parts := strings.SplitN(strings.TrimSpace(statusLine), " ", 3)
	if len(parts) < 2 {
		return 0
	}
	var code int
	fmt.Sscanf(parts[1], "%d", &code)
	return code
}

// ConnectErrorMeansDenied — обрыв/403 на CONNECT трактуем как отказ ACL.
func ConnectErrorMeansDenied(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "403") ||
		strings.Contains(s, "forbidden") ||
		strings.Contains(s, "eof") ||
		strings.Contains(s, "reset") ||
		strings.Contains(s, "aborted") ||
		strings.Contains(s, "timeout")
}

func looksForbiddenHTTP(status int, body string) bool {
	if status == http.StatusForbidden {
		return true
	}
	low := strings.ToLower(body)
	return strings.Contains(low, "forbidden") || strings.Contains(low, "заблокирован") || strings.Contains(low, "errordetail")
}

func proxyHTTPAllowed(proxyAddr string, in EvaluateInput, proxyUser, proxyPass string) (bool, error) {
	host := strings.TrimSpace(in.SNI)
	if host == "" {
		host = "localhost"
	}
	port := in.DstPort
	if port == 0 {
		port = 80
	}
	path := strings.TrimSpace(in.Path)
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	absoluteURL := fmt.Sprintf("http://%s:%d%s", host, port, path)

	ctx, cancel := context.WithTimeout(context.Background(), proxyProbeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, absoluteURL, nil)
	if err != nil {
		return false, err
	}
	token := base64.StdEncoding.EncodeToString([]byte(proxyUser + ":" + proxyPass))
	req.Header.Set("Proxy-Authorization", "Basic "+token)
	proxyURL, err := url.Parse("http://" + proxyAddr)
	if err != nil {
		return false, err
	}
	tr := &http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		DialContext:           (&net.Dialer{Timeout: proxyProbeTimeout}).DialContext,
		ResponseHeaderTimeout: proxyProbeTimeout,
	}
	client := &http.Client{Transport: tr, Timeout: proxyProbeTimeout}
	res, err := client.Do(req)
	if err != nil {
		low := strings.ToLower(err.Error())
		if strings.Contains(low, "403") || strings.Contains(low, "forbidden") {
			return false, nil
		}
		// origin недоступен — ACL уже пропустил
		return true, nil
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 256))
	if res.StatusCode == http.StatusForbidden {
		return false, nil
	}
	return true, nil
}

func dialCONNECTProbe(proxyAddr, dest, user, pass string) (status int, conn net.Conn, err error) {
	d := net.Dialer{Timeout: proxyProbeTimeout}
	conn, err = d.Dial("tcp", proxyAddr)
	if err != nil {
		return 0, nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(connectResponseTimeout))
	var req string
	if user != "" || pass != "" {
		token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		req = fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: Basic %s\r\n\r\n", dest, dest, token)
	} else {
		req = fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", dest, dest)
	}
	if _, err := io.WriteString(conn, req); err != nil {
		conn.Close()
		return 0, nil, err
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		conn.Close()
		return 0, nil, err
	}
	_, _ = io.ReadAll(io.LimitReader(resp.Body, 512))
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		conn.Close()
		return resp.StatusCode, nil, fmt.Errorf("CONNECT %s: HTTP %d", dest, resp.StatusCode)
	}
	_ = conn.SetDeadline(time.Time{})
	return resp.StatusCode, conn, nil
}
