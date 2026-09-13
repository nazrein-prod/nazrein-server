package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// reached is a stand-in for the real handler chain. If the validator lets a
// request through, this answers 200; anything else came from the validator.
func reached(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// validatorUnderTest mounts the spec validator in front of a catch-all, so a
// test can tell "the validator rejected this" from "the request was allowed".
func validatorUnderTest(t *testing.T) http.Handler {
	t.Helper()

	validate, err := specValidator()
	if err != nil {
		t.Fatalf("specValidator: %v", err)
	}
	return validate(http.HandlerFunc(reached))
}

func TestSpecValidator(t *testing.T) {
	h := validatorUnderTest(t)

	tests := []struct {
		name   string
		target string
		want   int
	}{
		// The regression this guards: the handler used to enforce
		// 1 <= limit <= 100 by hand. That check now lives only in the spec, so
		// if the validator is not wired, an unbounded LIMIT reaches ClickHouse.
		{"limit above maximum", "/api/v1/public/videos?limit=500", http.StatusBadRequest},
		{"limit at maximum", "/api/v1/public/videos?limit=100", http.StatusOK},
		{"limit below minimum", "/api/v1/public/videos?limit=0", http.StatusBadRequest},
		{"page below minimum", "/api/v1/public/videos?page=0", http.StatusBadRequest},
		{"page not a number", "/api/v1/public/videos?page=abc", http.StatusBadRequest},

		{"sortBy outside enum", "/api/v1/public/videos?sortBy=garbage", http.StatusBadRequest},
		{"sortBy in enum", "/api/v1/public/videos?sortBy=recent", http.StatusOK},
		{"type outside enum", "/api/v1/public/videos?type=nonsense", http.StatusBadRequest},

		// Every parameter is optional with a default.
		{"no parameters at all", "/api/v1/public/videos", http.StatusOK},
		{"all parameters", "/api/v1/public/videos?page=2&limit=25&sortBy=recent&type=channel&q=mrbeast", http.StatusOK},

		{"path id not a uuid", "/api/v1/public/videos/not-a-uuid", http.StatusBadRequest},
		{"path id is a uuid", "/api/v1/public/videos/6ba7b810-9dad-11d1-80b4-00c04fd430c8", http.StatusOK},

		{"autocomplete missing required q", "/api/v1/public/videos/autocomplete", http.StatusBadRequest},
		{"autocomplete q below minLength", "/api/v1/public/videos/autocomplete?q=a", http.StatusBadRequest},
		{"autocomplete q valid", "/api/v1/public/videos/autocomplete?q=ab", http.StatusOK},

		{"analytics limit above maximum", "/api/v1/public/videos/analytics/abc?limit=9999", http.StatusBadRequest},
		{"analytics offset negative", "/api/v1/public/videos/analytics/abc?offset=-1", http.StatusBadRequest},
		{"analytics valid", "/api/v1/public/videos/analytics/abc?limit=10&offset=20", http.StatusOK},

		// Browser-redirect OAuth routes are in the spec now; before they were
		// registered in chi but undocumented, so the validator would 404 them.
		{"user oauth login", "/auth/google/login", http.StatusOK},
		{"user oauth callback", "/auth/google/callback?code=x&state=y", http.StatusOK},
		{"admin oauth login", "/auth/admin/google/login", http.StatusOK},

		{"path not in spec", "/api/v1/does-not-exist", http.StatusNotFound},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.target, nil))

			if rec.Code != tc.want {
				t.Errorf("GET %s = %d, want %d\nbody: %s", tc.target, rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}

// Authenticated endpoints must reach the auth middleware rather than being
// rejected by the validator for having no credentials — the spec declares
// security, but enforcement belongs to Authenticate/AuthenticateAdmin.
func TestSpecValidator_DoesNotEnforceAuth(t *testing.T) {
	h := validatorUnderTest(t)

	for _, target := range []string{
		"/api/v1/dashboard/metrics",
		"/api/v1/videos/bookmarks",
		"/admin/request",
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want the validator to pass it through to the auth middleware\nbody: %s",
				target, rec.Code, rec.Body.String())
		}
	}
}

// A method the spec does not define for a known path must not reach a handler.
func TestSpecValidator_RejectsUndefinedMethod(t *testing.T) {
	h := validatorUnderTest(t)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/api/v1/public/videos", nil))

	if rec.Code == http.StatusOK {
		t.Error("DELETE /api/v1/public/videos was allowed; the spec defines only GET")
	}
}
