package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/grvbrk/track-yt-video/internal/models"
	"github.com/grvbrk/track-yt-video/internal/store"
	"github.com/grvbrk/track-yt-video/internal/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AdminOAuth interface {
	Login(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	Callback(w http.ResponseWriter, r *http.Request)
}

type AdminGoogleOauth struct {
	Logger    *log.Logger
	Config    *oauth2.Config
	Store     *sessions.CookieStore
	UserStore *store.PostgresUserStore
}

func NewAdminGoogleOauth(logger *log.Logger, adminStore *sessions.CookieStore, userStore *store.PostgresUserStore) (*AdminGoogleOauth, error) {
	return &AdminGoogleOauth{
		Logger: logger,
		Config: &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID_ADMIN"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET_ADMIN"),
			RedirectURL:  "http://localhost:8080/auth/admin/google/callback", // FIX
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.profile", "https://www.googleapis.com/auth/userinfo.email"},
			Endpoint:     google.Endpoint,
		},
		Store:     adminStore,
		UserStore: userStore,
	}, nil
}

func (g *AdminGoogleOauth) Login(w http.ResponseWriter, r *http.Request) {
	url := g.Config.AuthCodeURL("random-state-string", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (g *AdminGoogleOauth) Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := g.Store.Get(r, "session")
	delete(session.Values, "admin_email")
	session.Save(r, w)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (g *AdminGoogleOauth) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := g.Config.Exchange(context.Background(), code)
	if err != nil {
		g.Logger.Println("Error exchanging admin token", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}

	client := g.Config.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		g.Logger.Println("Error getting user info", err)
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
		g.Logger.Println("Error decoding user info", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}

	var userId string
	user, err := g.UserStore.GetUserByGoogleID(userInfo.GoogleID)
	if user == nil || err == sql.ErrNoRows {
		g.Logger.Println("User not found")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"Error": "Unauthorized"})
		return
	}

	if user.Role != "ADMIN" {
		g.Logger.Println("User not admin")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"Error": "Unauthorized"})
		return
	}

	userId = user.ID.String()

	session, _ := g.Store.Get(r, "session")
	session.Values["admin_email"] = userInfo.Email
	session.Values["admin_id"] = userId
	session.Values["admin_image"] = userInfo.Image
	session.Values["admin_name"] = userInfo.Name
	session.Options.Path = "/"
	// session.Options.Secure = false
	// session.Options.SameSite = http.SameSiteNoneMode

	err = session.Save(r, w)
	if err != nil {
		g.Logger.Println("Error saving admin session", err)
	}

	http.Redirect(w, r, "http://localhost:3001/dashboard", http.StatusSeeOther)
}

func (g *AdminGoogleOauth) AuthAdmin(w http.ResponseWriter, r *http.Request) {
	session, err := g.Store.Get(r, "session")
	if err != nil {
		g.Logger.Println("Failed to decode admin session:", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"Error": "Unauthorized"})
		return
	}

	email, ok := session.Values["admin_email"].(string)
	if !ok || email == "" {
		g.Logger.Println("No admin email found in session")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"Error": "Unauthorized"})
		return
	}

	adminID, _ := session.Values["admin_id"].(string)
	adminImage, _ := session.Values["admin_image"].(string)
	adminName, _ := session.Values["admin_name"].(string)
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"admin_id":    adminID,
		"admin_email": email,
		"admin_image": adminImage,
		"admin_name":  adminName,
	})
}

func (g *AdminGoogleOauth) GetAdmin(r *http.Request) (*models.User, error) {
	session, err := g.Store.Get(r, "session")

	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	adminEmail, ok := session.Values["admin_email"].(string)
	if !ok || adminEmail == "" {
		return nil, fmt.Errorf("no admin email found in session")
	}

	id, ok := session.Values["admin_id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("no admin id found in session")
	}

	adminID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("failed to parse admin id: %w", err)
	}

	return &models.User{
		ID:    adminID,
		Email: adminEmail,
	}, nil
}
