package hooks

import (
	"bytes"
	"crypto/tls"
	"io"
	stdhttp "net/http"

	"oktopus/internal/proxy/bytecount"
	"oktopus/internal/proxy/forbidden"
)

// WriteResponseConn пишет решение middleware в HTTP/1.1 соединение (MITM после TLS).
func (d Decision) WriteResponseConn(w io.Writer, req *stdhttp.Request) error {
	if !d.Handled() {
		return nil
	}
	res := d.buildResponse(req)
	if res == nil {
		return nil
	}
	if !d.Allow {
		if tc, ok := w.(*tls.Conn); ok {
			bytecount.MarkNextWriteDenied(tc.NetConn())
		}
	}
	return res.Write(w)
}

func (d Decision) buildResponse(req *stdhttp.Request) *stdhttp.Response {
	if d.Redirect != nil {
		return &stdhttp.Response{
			StatusCode: stdhttp.StatusTemporaryRedirect,
			Status:     "307 Temporary Redirect",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     stdhttp.Header{"Location": {d.Redirect.String()}},
			Body:       stdhttp.NoBody,
			Request:    req,
		}
	}
	if !d.Allow {
		body := forbidden.Snapshot()
		return &stdhttp.Response{
			StatusCode:    stdhttp.StatusForbidden,
			Status:        "403 Forbidden",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        stdhttp.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
			Body:          io.NopCloser(bytes.NewReader(body)),
			ContentLength: int64(len(body)),
			Request:       req,
		}
	}
	return nil
}
