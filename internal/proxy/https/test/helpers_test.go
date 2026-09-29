package https_test

import (
	"bufio"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"testing"
)

// dialCONNECT устанавливает туннель через HTTP-прокси к dest (host:port).
func dialCONNECT(t *testing.T, proxyAddr, dest string) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		t.Fatal(err)
	}
	req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", dest, dest)
	if _, err := io.WriteString(conn, req); err != nil {
		conn.Close()
		t.Fatal(err)
	}
	br := bufio.NewReader(conn)
	resp, err := stdhttp.ReadResponse(br, &stdhttp.Request{Method: stdhttp.MethodConnect})
	if err != nil {
		conn.Close()
		t.Fatal(err)
	}
	if resp.StatusCode != stdhttp.StatusOK {
		conn.Close()
		t.Fatalf("CONNECT status: %d", resp.StatusCode)
	}
	return &bufConn{Conn: conn, r: br}
}

type bufConn struct {
	net.Conn
	r *bufio.Reader
}

func (b *bufConn) Read(p []byte) (int, error) {
	return b.r.Read(p)
}
