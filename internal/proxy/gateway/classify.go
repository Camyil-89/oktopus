package gateway

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
	"syscall"
)

const (
	ErrTypeTimeout  = "timeout"
	ErrTypeDial     = "dial"
	ErrTypeTLS      = "tls"
	ErrTypeDNS      = "dns"
	ErrTypeCanceled = "canceled"
	ErrTypeUnknown  = "unknown"
)

// ClassifyError — категория upstream-ошибки для access log (gateway_error.type).
func ClassifyError(err error) string {
	if err == nil {
		return ErrTypeUnknown
	}
	if errors.Is(err, context.Canceled) {
		return ErrTypeCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return ErrTypeTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ErrTypeTimeout
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return ClassifyError(urlErr.Err)
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" {
			return ErrTypeDial
		}
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return ErrTypeDNS
	}
	var tlsRec tls.RecordHeaderError
	if errors.As(err, &tlsRec) {
		return ErrTypeTLS
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "tls:") || strings.Contains(msg, "x509:") || strings.Contains(msg, "certificate") {
		return ErrTypeTLS
	}
	if strings.Contains(msg, "dial tcp") || strings.Contains(msg, "connect:") || errors.Is(err, syscall.ECONNREFUSED) {
		return ErrTypeDial
	}
	if strings.Contains(msg, "no such host") || strings.Contains(msg, "lookup ") {
		return ErrTypeDNS
	}
	return ErrTypeUnknown
}
