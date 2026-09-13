package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/grvbrk/nazrein_server/internal/store"
	"github.com/grvbrk/nazrein_server/internal/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AdminOAuth interface {
	Login(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	Callback(w http.ResponseWriter, r *http.Request)
}

type AdminGoogleOauth struct {
	Logger    *slog.Logger
	Config    *oauth2.Config
	Store     *sessions.CookieStore
	UserStore *store.PostgresUserStore
}

func NewAdminGoogleOauth(logger *slog.Logger, adminStore *sessions.CookieStore, userStore *store.PostgresUserStore) (*AdminGoogleOauth, error) {
	return &AdminGoogleOauth{
		Logger: logger,
		Config: &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID_ADMIN"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET_ADMIN"),
			RedirectURL:  fmt.Sprintf("%s/auth/admin/google/callback", os.Getenv("NEXT_PUBLIC_BACKEND_URL")),
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.profile", "https://www.googleapis.com/auth/userinfo.email"},
			Endpoint:     google.Endpoint,
		},
		Store:     adminStore,
		UserStore: userStore,
	}, nil
}

func (g *AdminGoogleOauth) Login(w http.ResponseWriter, r *http.Request) {
	url, err := BeginOAuth(w, g.Config, AdminStateCookie)
	if err != nil {
		g.Logger.Error("Error starting admin oauth flow", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (g *AdminGoogleOauth) Callback(w http.ResponseWriter, r *http.Request) {
	verifier, err := CompleteOAuth(w, r, AdminStateCookie)
	if err != nil {
		g.Logger.Warn("Rejecting admin oauth callback", "err", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"Error": "Invalid login attempt"})
		return
	}

	code := r.URL.Query().Get("code")
	token, err := g.Config.Exchange(context.Background(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		g.Logger.Error("Error exchanging admin token", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}

	client := g.Config.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		g.Logger.Error("Error getting admin info", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}

	defer resp.Body.Close()

	var userInfo struct {
		GoogleID string `json:"id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Image    string `json:"picture"`
	}

	err = json.NewDecoder(resp.Body).Decode(&userInfo)
	if err != nil {
		g.Logger.Error("Error decoding user info", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}

	var userId string
	user, err := g.UserStore.GetUserByGoogleID(userInfo.GoogleID)
	if user == nil || err == sql.ErrNoRows {
		g.Logger.Warn("User not found")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"Error": "Unauthorized"})
		return
	}

	if user.Role != "ADMIN" {
		g.Logger.Warn("User not admin", "email", user.Email, "role", user.Role)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"Error": "Unauthorized"})
		return
	}

	userId = user.ID.String()

	session, _ := g.Store.Get(r, "nazrein_admin_session")
	session.Values["admin_email"] = userInfo.Email
	session.Values["admin_id"] = userId
	session.Values["admin_name"] = userInfo.Name
	session.Values["admin_image"] = userInfo.Image

	err = session.Save(r, w)
	if err != nil {
		g.Logger.Error("Error saving admin session", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}

	redirectURL := os.Getenv("ADMIN_FRONTEND_URL") + "/dashboard"
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (g *AdminGoogleOauth) Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := g.Store.Get(r, "nazrein_admin_session")

	for key := range session.Values {
		delete(session.Values, key)
	}

	session.Options.MaxAge = -1

	err := session.Save(r, w)
	if err != nil {
		g.Logger.Warn("Error saving admin session", "err", err)
	}

	redirectURL := os.Getenv("ADMIN_FRONTEND_URL")
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (g *AdminGoogleOauth) AuthAdmin(w http.ResponseWriter, r *http.Request) {
	session, err := g.Store.Get(r, "nazrein_admin_session")
	if err != nil {
		g.Logger.Warn("Failed to decode admin session", "err", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"Error": "Unauthorized"})
		return
	}

	adminEmail, emailOk := session.Values["admin_email"].(string)
	adminIDStr, idOk := session.Values["admin_id"].(string)
	adminName, nameOk := session.Values["admin_name"].(string)
	adminImage, imageOk := session.Values["admin_image"].(string)

	if !emailOk || !idOk || !nameOk || !imageOk || adminEmail == "" || adminIDStr == "" || adminName == "" || adminImage == "" {
		g.Logger.Warn("Invalid or missing admin data in session")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "Not Authenticated"})
		return
	}

	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		g.Logger.Warn("Invalid admin ID format in session", "err", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "Not Authenticated"})
		return
	}

	adminInfo := map[string]interface{}{
		"id":    adminID,
		"email": adminEmail,
		"name":  adminName,
		"image": adminImage,
		"role":  "ADMIN",
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": adminInfo})
}
