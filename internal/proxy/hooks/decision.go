package hooks

import (
	stdhttp "net/http"
	"net/url"

	"oktopus/internal/proxy/forbidden"
)

// Decision — результат middleware: разрешить трафик, оборвать или перенаправить клиента.
type Decision struct {
	Allow    bool
	Redirect *url.URL // nil — без редиректа
}

// AllowDecision разрешает продолжение без редиректа.
func AllowDecision() Decision {
	return Decision{Allow: true}
}

// DenyDecision запрещает подключение или запрос.
func DenyDecision() Decision {
	return Decision{Allow: false}
}

// RedirectDecision разрешает цепочку, но подменяет ответ редиректом на url.
func RedirectDecision(u *url.URL) Decision {
	if u == nil {
		return AllowDecision()
	}
	return Decision{Allow: true, Redirect: u}
}

// Handled — middleware уже принял решение (запрет или редирект).
func (d Decision) Handled() bool {
	return !d.Allow || d.Redirect != nil
}

// WriteResponse пишет 403 или 3xx клиенту. Возвращает true, если ответ уже отправлен.
func (d Decision) WriteResponse(w stdhttp.ResponseWriter, r *stdhttp.Request) bool {
	if d.Redirect != nil {
		stdhttp.Redirect(w, r, d.Redirect.String(), stdhttp.StatusTemporaryRedirect)
		return true
	}
	if !d.Allow {
		forbidden.Write403(w)
		return true
	}
	return false
}
