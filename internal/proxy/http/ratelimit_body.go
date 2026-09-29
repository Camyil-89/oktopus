package http

import "io"

type rateLimitBody struct {
	io.Reader
	closer io.Closer
}

func (r rateLimitBody) Close() error {
	if r.closer == nil {
		return nil
	}
	return r.closer.Close()
}
