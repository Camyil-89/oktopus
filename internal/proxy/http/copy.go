package http

import (
	"io"
	"sync"

)

const copyBufferSize = 32 * 1024

var copyBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, copyBufferSize)
		return &b
	},
}

// Copy copies from src to dst using a pooled buffer.
func Copy(dst io.Writer, src io.Reader) (int64, error) {
	buf := copyBufPool.Get().(*[]byte)
	defer copyBufPool.Put(buf)
	return copyBuffer(dst, src, *buf)
}

func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	var written int64
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
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
