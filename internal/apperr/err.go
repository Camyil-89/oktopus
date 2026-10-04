package apperr

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

// Error — ошибка с кодом для клиента API.
type Error struct {
	Code   Code
	Client bool
	cause  error
}

func (e *Error) Error() string {
	return string(e.Code)
}

func (e *Error) Unwrap() error {
	return e.cause
}

// New создаёт клиентскую ошибку с кодом.
func New(code Code) *Error {
	return &Error{Code: code, Client: true}
}

// Wrap помечает внутреннюю ошибку кодом (клиентская).
func Wrap(code Code, cause error) *Error {
	return &Error{Code: code, Client: true, cause: cause}
}

// Internal — серверная ошибка с кодом (редко; для единообразия ответа).
func Internal(code Code, cause error) *Error {
	return &Error{Code: code, Client: false, cause: cause}
}

// CodeOf возвращает код, если err (или обёртка) — *Error.
func CodeOf(err error) (Code, bool) {
	var ae *Error
	if errors.As(err, &ae) {
		return ae.Code, true
	}
	return "", false
}

// IsClient true, если ошибка предназначена для отображения пользователю (4xx).
func IsClient(err error) bool {
	var ae *Error
	if errors.As(err, &ae) {
		return ae.Client
	}
	return false
}

// PublicMessage — значение поля error в JSON: код или исходный текст.
func PublicMessage(err error) string {
	if err == nil {
		return ""
	}
	if code, ok := CodeOf(err); ok {
		return string(code)
	}
	return err.Error()
}

// IsListenBindError — не удалось занять адрес прослушивания (порт занят и т.п.).
func IsListenBindError(err error) bool {
	var op *net.OpError
	return errors.As(err, &op) && op.Op == "listen"
}

// MapListenBind переводит ошибку listen/bind в код API.
func MapListenBind(err error) error {
	if err == nil {
		return nil
	}
	if IsListenBindError(err) {
		return Wrap(ListenAddressInUse, err)
	}
	return err
}

// ProxyStartErrorMessage — значение proxy_start_error в статусе (код или текст).
func ProxyStartErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return PublicMessage(MapListenBind(err))
}

// MapProxyApply переводит ошибки hot-reload прокси в коды API.
func MapProxyApply(err error) error {
	if err == nil {
		return nil
	}
	if mapped := MapListenBind(err); mapped != err {
		return mapped
	}
	msg := err.Error()
	if strings.Contains(msg, "mitm requires CA") {
		return Wrap(ProxyMITMCAMissing, err)
	}
	if strings.Contains(msg, "unknown connect mode") {
		return Wrap(InvalidConnectMode, err)
	}
	return fmt.Errorf("%s", strings.TrimPrefix(msg, "apply proxy config: "))
}
