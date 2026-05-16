package middleware

import (
	"log/slog"
	"net/http"

	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/response"
)

// Recovery is a middleware that recovers from panics, logs the panic, and
// returns a HTTP 500 (Internal Server Error) status if possible.
func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("panic recovered", "error", err)

					appErr := errors.InternalServerError(nil)
					appErr.Message = "Internal server error"
					response.Error(w, appErr)
				}
			}()

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
