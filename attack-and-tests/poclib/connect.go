package poclib

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
)

// DialCONNECT устанавливает туннель; при auth — Proxy-Authorization.
func DialCONNECT(proxyAddr, dest, proxyUser, proxyPass string) (net.Conn, *bufio.Reader, int, error) {
	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		return nil, nil, 0, err
	}
	var req string
	if proxyUser != "" || proxyPass != "" {
		token := base64.StdEncoding.EncodeToString([]byte(proxyUser + ":" + proxyPass))
		req = fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: Basic %s\r\n\r\n", dest, dest, token)
	} else {
		req = fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", dest, dest)
	}
	if _, err := io.WriteString(conn, req); err != nil {
		conn.Close()
		return nil, nil, 0, err
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		conn.Close()
		return nil, nil, 0, err
	}
	code := resp.StatusCode
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if code != http.StatusOK {
		conn.Close()
		return nil, nil, code, fmt.Errorf("CONNECT %s: HTTP %d", dest, code)
	}
	return conn, br, code, nil
}
