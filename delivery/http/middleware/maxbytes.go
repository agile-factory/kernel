package middleware

import "net/http"

// DefaultMaxBodyBytes caps request bodies when ApplyGlobalMiddlewares is used.
const DefaultMaxBodyBytes int64 = 1 << 20 // 1 MiB

// MaxBytes limits request bodies to n bytes. Reading past the limit fails
// (json.Decoder returns an error, handlers answer 400) instead of letting a
// client stream an unbounded body into memory.
func MaxBytes(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, n)
			}
			next.ServeHTTP(w, r)
		})
	}
}
