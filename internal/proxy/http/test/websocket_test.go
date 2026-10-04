package http_test

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"golang.org/x/net/websocket"

	proxyhttp "oktopus/internal/proxy/http"
)

func TestForwarderWebSocketUpgrade(t *testing.T) {
	origin := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		var msg string
		if err := websocket.Message.Receive(ws, &msg); err != nil {
			return
		}
		_ = websocket.Message.Send(ws, "echo:"+msg)
	}))
	defer origin.Close()

	originURL, err := url.Parse(origin.URL)
	if err != nil {
		t.Fatal(err)
	}

	f := &proxyhttp.Forwarder{Client: origin.Client()}
	proxy := httptest.NewUnstartedServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		r.URL.Scheme = originURL.Scheme
		r.URL.Host = originURL.Host
		if err := f.Serve(r.Context(), w, r); err != nil {
			stdhttp.Error(w, err.Error(), stdhttp.StatusBadGateway)
		}
	}))
	proxy.Start()
	defer proxy.Close()

	conn, err := net.Dial("tcp", proxy.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	secKey := base64.StdEncoding.EncodeToString(key)
	absolute := origin.URL
	req := fmt.Sprintf(
		"GET %s HTTP/1.1\r\nHost: %s\r\nOrigin: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		absolute, originURL.Host, origin.URL, secKey,
	)
	if _, err := io.WriteString(conn, req); err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(conn)
	res, err := stdhttp.ReadResponse(br, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != stdhttp.StatusSwitchingProtocols {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		res.Body.Close()
		t.Fatalf("status %d body %q", res.StatusCode, body)
	}
	res.Body.Close()

	if err := writeClientTextFrame(conn, "hi"); err != nil {
		t.Fatal(err)
	}
	got, err := readServerTextFrame(br)
	if err != nil {
		t.Fatal(err)
	}
	if got != "echo:hi" {
		t.Fatalf("payload: %q", got)
	}
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
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}
