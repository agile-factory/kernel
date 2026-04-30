package response

import (
	"encoding/json"
	"net/http"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
)

// JSON sends a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			// Cannot do much here since headers are already written
			_ = err
		}
	}
}

// Error sends a JSON error response based on the provided error.
// It formats it cleanly for the API consumer.
func Error(w http.ResponseWriter, err error) {
	var appErr *errors.AppError
	if e, ok := err.(*errors.AppError); ok {
		appErr = e
	} else {
		appErr = errors.InternalServerError(err)
	}

	resp := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    appErr.Code,
			"message": appErr.Message,
		},
	}

	JSON(w, appErr.HTTPCode, resp)
}
