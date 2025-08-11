package middlewares

import (
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/grvbrk/nazrein_server/internal/utils"
)

var allowedOrigins = []string{
	"http://localhost:3000",
	"http://localhost:3001",
}

type MiddlwareHandler struct {
	SessionStore *sessions.CookieStore
	logger       *log.Logger
}

func NewMiddlewareHandler(logger *log.Logger, store *sessions.CookieStore) *MiddlwareHandler {
	return &MiddlwareHandler{
		logger:       logger,
		SessionStore: store,
	}
}

func (mh *MiddlwareHandler) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Check the session in gorilla session store
		// If exists, allow the request to pass
		// If not, return 401

		session, err := mh.SessionStore.Get(r, "session")
		if err != nil {
			mh.logger.Println("No session found in auth middleware")
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "Not Authorized"})
			return
		}

		userEmail, ok := session.Values["user_email"].(string)
		if !ok || userEmail == "" {
			mh.logger.Println("No user/user_email found in auth middleware")
			utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "Not Authorized"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (mh *MiddlwareHandler) Cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if !isOriginAllowed(origin) {
			mh.logger.Println("Not allowed origin", origin)
			utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// preflight (OPTIONS)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (mh *MiddlwareHandler) RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		mh.logger.Println("Origin", origin)
		mh.logger.Printf("Incoming request: %s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func isOriginAllowed(origin string) bool {
	for _, allowedOrigin := range allowedOrigins {
		if origin == allowedOrigin {
			return true
		}
	}
	return false
}
