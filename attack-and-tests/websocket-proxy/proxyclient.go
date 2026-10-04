package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
)

func dialCONNECT(proxyAddr, dest, proxyUser, proxyPass string) (net.Conn, *bufio.Reader, error) {
	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	br := bufio.NewReader(conn)
	resp, err := stdhttp.ReadResponse(br, &stdhttp.Request{Method: stdhttp.MethodConnect})
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	if resp.StatusCode != stdhttp.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		conn.Close()
		return nil, nil, fmt.Errorf("CONNECT %s: HTTP %d", dest, resp.StatusCode)
	}
	resp.Body.Close()
	return conn, br, nil
}

type bufConn struct {
	net.Conn
	r *bufio.Reader
}

func (b *bufConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}
