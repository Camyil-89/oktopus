package bytecount

import (
	"bytes"
	"crypto/tls"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/google/uuid"
)

// clientConn считает байты на границе прокси↔клиент (как в CONNECT tunnel).
type clientConn struct {
	net.Conn
	instanceID    uuid.UUID
	sessionDenied bool
	nextWriteDenied atomic.Bool
}

// WrapConn — Read с TCP клиента: исх, Write на TCP клиента: вх (как в CONNECT tunnel).
// Для MITM оборачивать соединение до tls.Server.
func WrapConn(c net.Conn) net.Conn {
	return WrapConnPolicy(c, false)
}

// WrapConnPolicy — как WrapConn; sessionDenied помечает весь поток (forbidden MITM).
func WrapConnPolicy(c net.Conn, sessionDenied bool) net.Conn {
	return WrapConnPolicyInstance(c, sessionDenied, uuid.Nil)
}

// WrapConnPolicyInstance — как WrapConnPolicy, с привязкой к proxy-инстансу.
func WrapConnPolicyInstance(c net.Conn, sessionDenied bool, instanceID uuid.UUID) net.Conn {
	if c == nil {
		return c
	}
	return &clientConn{Conn: c, instanceID: instanceID, sessionDenied: sessionDenied}
}

func (c *clientConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		observeUpInstance(c.instanceID, n, c.sessionDenied)
	}
	return n, err
}

func (c *clientConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 {
		denied := c.sessionDenied || c.nextWriteDenied.Load()
		c.nextWriteDenied.Store(false)
		observeDownInstance(c.instanceID, n, denied)
	}
	return n, err
}

// MarkNextWriteDenied помечает следующую запись на соединении (403 внутри allow MITM).
func MarkNextWriteDenied(conn net.Conn) {
	cc := findClientConn(conn)
	if cc != nil {
		cc.nextWriteDenied.Store(true)
	}
}

func findClientConn(c net.Conn) *clientConn {
	for c != nil {
		if cc, ok := c.(*clientConn); ok {
			return cc
		}
		switch v := c.(type) {
		case *tls.Conn:
			c = v.NetConn()
		default:
			return nil
		}
	}
	return nil
}

type respWriter struct {
	http.ResponseWriter
	instanceID uuid.UUID
	denied     bool
}

// WrapResponseWriter считает тело/заголовки ответа клиенту (plain HTTP proxy).
func WrapResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	return WrapResponseWriterPolicy(w, false)
}

// WrapResponseWriterPolicy — как WrapResponseWriter, с учётом allow/deny.
func WrapResponseWriterPolicy(w http.ResponseWriter, denied bool) http.ResponseWriter {
	return WrapResponseWriterPolicyInstance(w, denied, uuid.Nil)
}

// WrapResponseWriterPolicyInstance — как WrapResponseWriterPolicy, с привязкой к инстансу.
func WrapResponseWriterPolicyInstance(w http.ResponseWriter, denied bool, instanceID uuid.UUID) http.ResponseWriter {
	if w == nil {
		return w
	}
	return respWriter{ResponseWriter: w, instanceID: instanceID, denied: denied}
}

func (w respWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if n > 0 {
		observeDownInstance(w.instanceID, n, w.denied)
	}
	return n, err
}

// ResponseWriterInstrumented — обёртка bytecount уже считает ответ.
func ResponseWriterInstrumented(w http.ResponseWriter) bool {
	_, ok := w.(respWriter)
	return ok
}

// SupplementDeniedDown — учёт 403, если ResponseWriter не обёрнут bytecount.
func SupplementDeniedDown(w http.ResponseWriter, n int) {
	if n <= 0 || ResponseWriterInstrumented(w) {
		return
	}
	observeDown(n, true)
}

// ObserveRequestLineHeaders — исх: стартовая строка и заголовки входящего запроса (без тела).
func ObserveRequestLineHeaders(r *http.Request) {
	ObserveRequestLineHeadersInstance(r, uuid.Nil, false)
}

// ObserveRequestLineHeadersDenied — то же для заблокированного запроса.
func ObserveRequestLineHeadersDenied(r *http.Request) {
	ObserveRequestLineHeadersInstance(r, uuid.Nil, true)
}

// ObserveRequestLineHeadersInstance — стартовая строка и заголовки с привязкой к инстансу.
func ObserveRequestLineHeadersInstance(r *http.Request, instanceID uuid.UUID, denied bool) {
	if r == nil || r.URL == nil {
		return
	}
	var b bytes.Buffer
	b.WriteString(r.Method)
	b.WriteByte(' ')
	b.WriteString(r.URL.String())
	b.WriteString("\r\n")
	_ = r.Header.Write(&b)
	observeUpInstance(instanceID, b.Len(), denied)
}
