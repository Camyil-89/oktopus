package service

import (
	"errors"

	"oktopus/internal/proxy/acl/squid"
)

func diagnosticsFromCompileErr(err error) []squid.Diagnostic {
	var ce *squid.CompileErrors
	if errors.As(err, &ce) {
		return ce.Diagnostics
	}
	if err == nil {
		return nil
	}
	return []squid.Diagnostic{{
		Line:     0,
		Severity: squid.SeverityError,
		Code:     "compile",
		Message:  err.Error(),
	}}
}
