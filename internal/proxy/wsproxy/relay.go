package wsproxy

import (
	"bufio"
	"io"
	"net"
	"sync"

	"oktopus/internal/proxy/bytecount"
	"oktopus/internal/proxy/ratelimit"
	"oktopus/internal/proxy/wsactive"
)

// RelayLegs — клиентская сторона после ответа origin (101 или ошибка).
type RelayLegs struct {
	ClientWrite net.Conn
	ClientRead  io.Reader
	Switching   bool
}

type relayEndpoint struct {
	r io.Reader
	c io.Closer
}

func (e *relayEndpoint) Read(p []byte) (int, error) {
	return e.r.Read(p)
}

func (e *relayEndpoint) Close() error {
	if e.c == nil {
		return nil
	}
	return e.c.Close()
}

const relayBufferSize = 32 * 1024

var relayBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, relayBufferSize)
		return &b
	},
}

// RelayPair проксирует WebSocket-фреймы до закрытия одной из сторон.
func RelayPair(clientWrite net.Conn, clientRead io.Reader, upstream net.Conn, upstreamBR *bufio.Reader, flow *ratelimit.Flow) {
	wsactive.Inc()
	defer wsactive.Dec()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		relayWS(upstream, &relayEndpoint{r: clientRead, c: clientWrite}, flow, true)
		wg.Done()
	}()
	go func() {
		relayWS(clientWrite, &relayEndpoint{r: upstreamBR, c: upstream}, flow, false)
		wg.Done()
	}()
	wg.Wait()
}

// relayWS: fromClient=true — байты client→upstream (исх).
func relayWS(dst io.WriteCloser, src io.ReadCloser, flow *ratelimit.Flow, fromClient bool) {
	defer dst.Close()
	defer src.Close()
	buf := relayBufPool.Get().(*[]byte)
	defer relayBufPool.Put(buf)
	for {
		nr, er := src.Read(*buf)
		if nr > 0 {
			if fromClient {
				bytecount.ObserveUp(nr)
			} else {
				bytecount.ObserveDown(nr)
			}
			if flow != nil {
				if err := flow.Acquire(nr); err != nil {
					return
				}
			}
			nw, ew := dst.Write((*buf)[:nr])
			if ew != nil || (nw > 0 && nw < nr) {
				return
			}
		}
		if er != nil {
			return
		}
	}
}
