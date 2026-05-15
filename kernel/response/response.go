// Package response writes JSON success and error envelopes for the HTTP
// delivery layer. Error always works through *errors.AppError: any error
// passed in is unwrapped via errors.As, so wrapped AppErrors keep their
// code, message and HTTP status all the way to the client.
//
// In development the response body includes the originating stack frame
// under "debug" so a caller can jump straight to the source without
// scraping server logs. In staging/production that field is suppressed —
// only Code, Message and TraceID reach the wire.
//
// Usage:
//
//	if err := uc.Execute(ctx, params); err != nil {
//	    response.Error(w, err)
//	    return
//	}
//	response.JSON(w, http.StatusOK, dto)
package response

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"os"

	"github.com/AgileFactory/kernel/kernel/errors"
)

// JSON sends a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			_ = err
		}
	}
}

// errorBody is the JSON shape returned to API consumers.
// Stack and internal cause are intentionally absent — they live in logs only.
type errorBody struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	TraceID  string `json:"trace_id,omitempty"`
	Debug    string `json:"debug,omitempty"` // populated in non-production only
}

// isProduction returns true when APP_ENV is "production" or "staging".
// Both environments suppress internal stack details from API responses.
func isProduction() bool {
	env := os.Getenv("APP_ENV")
	return env == "production" || env == "staging"
}

// Error sends a JSON error response.
//
// Behaviour:
//   - Uses errors.As to unwrap the chain — works for wrapped AppErrors.
//   - In production/staging: only Code and Message reach the client.
//   - In development: also includes a "debug" field with the first stack frame
//     so developers can pinpoint the call site without reading server logs.
//   - TraceID is always forwarded when present (enables client-side log lookup).
//   - The raw Cause and full stack trace are NEVER serialised into the response.
func Error(w http.ResponseWriter, err error) {
	var appErr *errors.AppError

	// errors.As walks the entire Unwrap() chain, so wrapped AppErrors are found
	// even when the top-level value is a plain error.
	if !stderrors.As(err, &appErr) {
		appErr = errors.InternalServerError(err)
	}

	body := errorBody{
		Code:    appErr.Code,
		Message: appErr.Message,
		TraceID: appErr.TraceID,
	}

	// In non-production environments, surface the most useful stack frame so
	// developers can jump directly to the origin without opening server logs.
	if !isProduction() && len(appErr.Stack) > 0 {
		body.Debug = appErr.Stack[0].ShortFile() + ":" +
			itoa(appErr.Stack[0].Line) + " · " +
			appErr.Stack[0].ShortFunction()
	}

	JSON(w, appErr.HTTPCode, map[string]interface{}{"error": body})
}

// itoa converts an int to its decimal string — avoids importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte(n%10) + '0'
		n /= 10
	}
	return string(buf[pos:])
}
