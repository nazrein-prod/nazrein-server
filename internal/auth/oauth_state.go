package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	stateCookieTTL   = 10 * time.Minute
	UserStateCookie  = "nazrein_oauth_state"
	AdminStateCookie = "nazrein_admin_oauth_state"
)

// randomURLSafe returns n bytes of crypto-random data, base64url encoded.
func randomURLSafe(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// BeginOAuth mints a state value and a PKCE verifier, stores both in a
// short-lived cookie, and returns the provider URL to redirect the user to.
func BeginOAuth(w http.ResponseWriter, cfg *oauth2.Config, cookieName string) (string, error) {
	state, err := randomURLSafe(32)
	if err != nil {
		return "", err
	}
	verifier := oauth2.GenerateVerifier()

	secure := os.Getenv("ENV") == "production"
	http.SetCookie(w, &http.Cookie{
		Name:  cookieName,
		Value: state + "." + verifier,
		Path:  "/",
		// The provider redirects back as a top-level GET, which Lax still
		// carries — and Lax keeps the value off cross-site subrequests.
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		Secure:   secure,
		MaxAge:   int(stateCookieTTL.Seconds()),
	})

	return cfg.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	), nil
}

// CompleteOAuth verifies the returned state against the cookie and returns the
// PKCE verifier for the token exchange. The cookie is always cleared, so a
// state value is good for exactly one attempt.
func CompleteOAuth(w http.ResponseWriter, r *http.Request, cookieName string) (verifier string, err error) {
	// Whatever happens, this attempt is spent.
	defer clearStateCookie(w, cookieName)

	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", fmt.Errorf("missing or expired oauth state cookie: %w", err)
	}

	storedState, storedVerifier, ok := strings.Cut(cookie.Value, ".")
	if !ok || storedState == "" || storedVerifier == "" {
		return "", fmt.Errorf("malformed oauth state cookie")
	}

	returnedState := r.URL.Query().Get("state")
	if returnedState == "" {
		return "", fmt.Errorf("missing state in callback")
	}

	// Constant-time so the comparison leaks nothing about the expected value.
	if subtle.ConstantTimeCompare([]byte(storedState), []byte(returnedState)) != 1 {
		return "", fmt.Errorf("oauth state mismatch")
	}

	return storedVerifier, nil
}

func clearStateCookie(w http.ResponseWriter, cookieName string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		Secure:   os.Getenv("ENV") == "production",
		MaxAge:   -1,
	})
}
