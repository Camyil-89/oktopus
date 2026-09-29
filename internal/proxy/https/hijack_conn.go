package https

import (
	"bufio"
	"net"
)

// hijackedConn читает ClientHello из bufio (уже в буфере), пишет TLS-ответы сразу в TCP.
// Если писать в bufio.Writer, handshake зависает до Flush → ERR_TIMED_OUT в браузере.
type hijackedConn struct {
	net.Conn
	reader *bufio.Reader
}

func newHijackedConn(c net.Conn, rw *bufio.ReadWriter) net.Conn {
	return &hijackedConn{Conn: c, reader: rw.Reader}
}

func (c *hijackedConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}
