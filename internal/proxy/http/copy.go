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
	return io.CopyBuffer(dst, src, *buf)
}
