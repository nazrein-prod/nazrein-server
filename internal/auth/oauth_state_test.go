package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func testConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:    "test-client",
		RedirectURL: "https://api.example.test/auth/google/callback",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://accounts.google.test/o/oauth2/auth",
			TokenURL: "https://oauth2.google.test/token",
		},
	}
}

// begin runs BeginOAuth and returns the redirect URL plus the cookie it set.
func begin(t *testing.T) (*url.URL, *http.Cookie) {
	t.Helper()

	rec := httptest.NewRecorder()
	raw, err := BeginOAuth(rec, testConfig(), UserStateCookie)
	if err != nil {
		t.Fatalf("BeginOAuth: %v", err)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parsing auth URL: %v", err)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected exactly one cookie, got %d", len(cookies))
	}
	return parsed, cookies[0]
}

func TestBeginOAuth_SetsStateAndPKCE(t *testing.T) {
	authURL, cookie := begin(t)

	state := authURL.Query().Get("state")
	if state == "" {
		t.Fatal("no state on the authorization URL")
	}
	if state == "random-state-string" {
		t.Fatal("state is the old hardcoded literal")
	}
	if len(state) < 32 {
		t.Errorf("state is only %d chars; want a high-entropy value", len(state))
	}

	if got := authURL.Query().Get("code_challenge_method"); got != "S256" {
		t.Errorf("code_challenge_method = %q, want S256", got)
	}
	if authURL.Query().Get("code_challenge") == "" {
		t.Error("no PKCE code_challenge on the authorization URL")
	}

	if !cookie.HttpOnly {
		t.Error("state cookie must be HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Error("state cookie must be SameSite=Lax so it survives the provider redirect")
	}
	if !strings.HasPrefix(cookie.Value, state+".") {
		t.Error("cookie does not carry the state that was sent to the provider")
	}
}

func TestBeginOAuth_StateIsUniquePerAttempt(t *testing.T) {
	first, _ := begin(t)
	second, _ := begin(t)

	if first.Query().Get("state") == second.Query().Get("state") {
		t.Fatal("two login attempts produced the same state")
	}
}

// callback builds a callback request carrying the given cookie and state.
func callback(cookie *http.Cookie, state string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/auth/google/callback?code=abc&state="+url.QueryEscape(state), nil)
	if cookie != nil {
		r.AddCookie(cookie)
	}
	return r
}

func TestCompleteOAuth_AcceptsMatchingState(t *testing.T) {
	authURL, cookie := begin(t)
	state := authURL.Query().Get("state")

	verifier, err := CompleteOAuth(httptest.NewRecorder(), callback(cookie, state), UserStateCookie)
	if err != nil {
		t.Fatalf("CompleteOAuth rejected a valid callback: %v", err)
	}
	if verifier == "" {
		t.Error("no PKCE verifier returned for the token exchange")
	}
	if !strings.HasSuffix(cookie.Value, verifier) {
		t.Error("returned verifier is not the one stored at login")
	}
}

// The core regression: a callback whose state does not match the cookie is the
// shape of a login-CSRF attack and must be refused.
func TestCompleteOAuth_RejectsBadCallbacks(t *testing.T) {
	authURL, cookie := begin(t)
	goodState := authURL.Query().Get("state")

	tests := []struct {
		name   string
		cookie *http.Cookie
		state  string
	}{
		{"attacker-supplied state", cookie, "attacker-controlled-state"},
		{"old hardcoded literal", cookie, "random-state-string"},
		{"no state in callback", cookie, ""},
		{"no cookie at all", nil, goodState},
		{"malformed cookie", &http.Cookie{Name: UserStateCookie, Value: "no-separator"}, goodState},
		{"empty verifier half", &http.Cookie{Name: UserStateCookie, Value: goodState + "."}, goodState},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := CompleteOAuth(httptest.NewRecorder(), callback(tc.cookie, tc.state), UserStateCookie); err == nil {
				t.Error("callback was accepted; expected rejection")
			}
		})
	}
}

func TestCompleteOAuth_StateIsSingleUse(t *testing.T) {
	authURL, cookie := begin(t)
	state := authURL.Query().Get("state")

	rec := httptest.NewRecorder()
	if _, err := CompleteOAuth(rec, callback(cookie, state), UserStateCookie); err != nil {
		t.Fatalf("first use should succeed: %v", err)
	}

	// The response must clear the cookie, so a replayed callback finds nothing.
	cleared := rec.Result().Cookies()
	if len(cleared) != 1 || cleared[0].MaxAge >= 0 {
		t.Fatal("state cookie was not cleared after use")
	}
}

// The two flows must not validate against each other's cookie.
func TestCompleteOAuth_FlowsAreIsolated(t *testing.T) {
	authURL, userCookie := begin(t)
	state := authURL.Query().Get("state")

	if _, err := CompleteOAuth(httptest.NewRecorder(), callback(userCookie, state), AdminStateCookie); err == nil {
		t.Error("a user-flow cookie satisfied the admin flow")
	}
}
