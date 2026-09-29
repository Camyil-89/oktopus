package https

import (
	"io"
	"sync"

	"oktopus/internal/proxy/bytecount"
	"oktopus/internal/proxy/ratelimit"
)

// RelayDir — направление относительно клиента прокси.
type RelayDir int

const (
	RelayFromClient RelayDir = 0 // client → upstream (исх)
	RelayToClient   RelayDir = 1 // upstream → client (вх)
)

const relayBufferSize = 32 * 1024

var relayBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, relayBufferSize)
		return &b
	},
}

// Relay копирует байты src → dst и закрывает оба конца.
// flow != nil — один Acquire на chunk (без TCP ReadFrom/WriteTo, симметричный up/down).
// countBytes — учёт на границе tunnel; для MITM/WebSocket с bytecount.WrapConn — false.
func Relay(dst io.WriteCloser, src io.ReadCloser, flow *ratelimit.Flow, dir RelayDir, countBytes bool) {
	defer dst.Close()
	defer src.Close()
	buf := relayBufPool.Get().(*[]byte)
	defer relayBufPool.Put(buf)
	if flow == nil {
		_, _ = relayCopy(dst, src, nil, *buf, dir, countBytes)
		return
	}
	_, _ = relayCopy(dst, src, flow, *buf, dir, countBytes)
}

func relayCopy(dst io.Writer, src io.Reader, flow *ratelimit.Flow, buf []byte, dir RelayDir, countBytes bool) (int64, error) {
	var written int64
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			if countBytes {
				switch dir {
				case RelayFromClient:
					bytecount.ObserveUp(nr)
				case RelayToClient:
					bytecount.ObserveDown(nr)
				}
			}
			if flow != nil {
				if err := flow.Acquire(nr); err != nil {
					return written, err
				}
			}
			nw, ew := dst.Write(buf[:nr])
			written += int64(nw)
			if nw < nr && ew == nil {
				ew = io.ErrShortWrite
			}
			if ew != nil {
				return written, ew
			}
		}
		if er != nil {
			if er == io.EOF {
				return written, nil
			}
			return written, er
		}
	}
}
