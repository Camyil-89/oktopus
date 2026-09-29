package https

import (
	"bufio"
	stdhttp "net/http"
	"strings"
	"testing"
)

func TestIsWebSocketUpgrade_readRequest(t *testing.T) {
	raw := "GET / HTTP/1.1\r\nHost: example.com\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n"
	req, err := stdhttp.ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatal(err)
	}
	if !isWebSocketUpgrade(req) {
		t.Fatalf("headers: %v", req.Header)
	}
}
