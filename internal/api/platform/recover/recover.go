package recover

import (
	"log"
	"net/http"
	"runtime/debug"
)

// Middleware логирует панику обработчика в консоль и отвечает 500.
func Middleware(logger *log.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Printf("api: panic in request handler: %v\n%s", rec, debug.Stack())
					http.Error(w, "internal error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
