package poclib

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
)

// DialCONNECTNoAuth — CONNECT без Proxy-Authorization.
func DialCONNECTNoAuth(proxyAddr, dest string) (int, error) {
	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		return 0, err
	}
	req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", dest, dest)
	if _, err := io.WriteString(conn, req); err != nil {
		conn.Close()
		return 0, err
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		conn.Close()
		return 0, err
	}
	code := resp.StatusCode
	resp.Body.Close()
	conn.Close()
	return code, nil
}
