package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"time"
)

const proxyIOTimeout = 15 * time.Second

func dialCONNECT(proxyAddr, dest, proxyUser, proxyPass string) (status int, conn net.Conn, br *bufio.Reader, err error) {
	d := net.Dialer{Timeout: proxyIOTimeout}
	conn, err = d.Dial("tcp", proxyAddr)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("прокси %s: %w", proxyAddr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(proxyIOTimeout))

	var req string
	if proxyUser != "" || proxyPass != "" {
		token := base64.StdEncoding.EncodeToString([]byte(proxyUser + ":" + proxyPass))
		req = fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: Basic %s\r\n\r\n", dest, dest, token)
	} else {
		req = fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", dest, dest)
	}
	if _, err := io.WriteString(conn, req); err != nil {
		conn.Close()
		return 0, nil, nil, err
	}
	br = bufio.NewReader(conn)
	resp, err := stdhttp.ReadResponse(br, &stdhttp.Request{Method: stdhttp.MethodConnect})
	if err != nil {
		conn.Close()
		return 0, nil, nil, fmt.Errorf("ответ CONNECT: %w", err)
	}
	bodyPreview, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	resp.Body.Close()
	if resp.StatusCode != stdhttp.StatusOK {
		conn.Close()
		return resp.StatusCode, nil, nil, fmt.Errorf("CONNECT %s: HTTP %d %s", dest, resp.StatusCode, string(bodyPreview))
	}
	_ = conn.SetDeadline(time.Time{})
	return resp.StatusCode, conn, br, nil
}
