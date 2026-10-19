package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/agile-factory/kernel/kernel/errors"
)

func renderError(t *testing.T) map[string]any {
	t.Helper()
	w := httptest.NewRecorder()
	Error(w, errors.BadRequest("nope"))
	var out struct {
		Error map[string]any `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.Error
}

func TestError_DebugIsOptIn(t *testing.T) {
	for env, want := range map[string]bool{
		"": false, "production": false, "staging": false, "prod": false, "dev": false,
		"development": true, "test": true,
	} {
		t.Setenv("APP_ENV", env)
		_, has := renderError(t)["debug"]
		if has != want {
			t.Errorf("APP_ENV=%q: debug present = %v, want %v", env, has, want)
		}
	}
}
