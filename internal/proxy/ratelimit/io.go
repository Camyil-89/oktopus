package ratelimit

import (
	"io"
	"net"
	"time"
)

// WrapConn ограничивает Read и Write (для точечной обёртки; CONNECT — Relay(..., flow)).
func (f *Flow) WrapConn(c net.Conn) net.Conn {
	if f == nil || c == nil {
		return c
	}
	return &throttledConn{c: c, flow: f}
}

// Reader оборачивает io.Reader.
func (f *Flow) Reader(r io.Reader) io.Reader {
	if f == nil || r == nil {
		return r
	}
	return &throttledReader{r: r, flow: f}
}

// Writer оборачивает io.Writer.
func (f *Flow) Writer(w io.Writer) io.Writer {
	if f == nil || w == nil {
		return w
	}
	return &throttledWriter{w: w, flow: f}
}

type throttledConn struct {
	c    net.Conn
	flow *Flow
}

func (c *throttledConn) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return c.c.Read(p)
	}
	if err := c.flow.Acquire(len(p)); err != nil {
		return 0, err
	}
	return c.c.Read(p)
}

func (c *throttledConn) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return c.c.Write(p)
	}
	if err := c.flow.Acquire(len(p)); err != nil {
		return 0, err
	}
	return c.c.Write(p)
}

func (c *throttledConn) Close() error                       { return c.c.Close() }
func (c *throttledConn) LocalAddr() net.Addr                { return c.c.LocalAddr() }
func (c *throttledConn) RemoteAddr() net.Addr              { return c.c.RemoteAddr() }
func (c *throttledConn) SetDeadline(t time.Time) error      { return c.c.SetDeadline(t) }
func (c *throttledConn) SetReadDeadline(t time.Time) error  { return c.c.SetReadDeadline(t) }
func (c *throttledConn) SetWriteDeadline(t time.Time) error { return c.c.SetWriteDeadline(t) }

type throttledReader struct {
	r    io.Reader
	flow *Flow
}

func (r *throttledReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		if werr := r.flow.Acquire(n); werr != nil {
			return n, werr
		}
	}
	return n, err
}

type throttledWriter struct {
	w    io.Writer
	flow *Flow
}

func (w *throttledWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return w.w.Write(p)
	}
	if err := w.flow.Acquire(len(p)); err != nil {
		return 0, err
	}
	return w.w.Write(p)
}
