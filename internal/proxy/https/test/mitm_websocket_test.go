package https_test

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/websocket"

	proxyhttps "oktopus/internal/proxy/https"
)

func TestMITMWebSocketUpgrade(t *testing.T) {
	if testing.Short() {
		t.Skip("MITM integration")
	}

	origin := httptest.NewTLSServer(websocket.Handler(func(ws *websocket.Conn) {
		var msg string
		if err := websocket.Message.Receive(ws, &msg); err != nil {
			return
		}
		_ = websocket.Message.Send(ws, "echo:"+msg)
	}))
	defer origin.Close()

	originHost := origin.Listener.Addr().String()
	ca, pool := testCA(t)
	mitm := &proxyhttps.MITM{
		CA:                ca,
		OutboundTransport: origin.Client().Transport,
	}
	proxyAddr := startCONNECTProxy(t, mitm)

	conn := tls.Client(
		dialCONNECT(t, proxyAddr, originHost),
		&tls.Config{
			RootCAs:    pool,
			ServerName: "127.0.0.1",
			MinVersion: tls.VersionTLS12,
		},
	)
	defer conn.Close()
	if err := conn.Handshake(); err != nil {
		t.Fatal(err)
	}

	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	secKey := base64.StdEncoding.EncodeToString(key)
	req := fmt.Sprintf(
		"GET / HTTP/1.1\r\nHost: %s\r\nOrigin: https://%s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		originHost, originHost, secKey,
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
		t.Fatalf("status %d body %q headers %v", res.StatusCode, body, res.Header)
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

	var hdr []byte
	switch {
	case len(p) <= 125:
		hdr = []byte{0x81, byte(0x80 | len(p))}
	default:
		return fmt.Errorf("payload too long for test helper")
	}
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
