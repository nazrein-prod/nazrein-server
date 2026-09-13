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
	"github.com/grvbrk/nazrein_server/internal/models"
	"github.com/grvbrk/nazrein_server/internal/store"
	"github.com/grvbrk/nazrein_server/internal/utils"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type Oauth interface {
	Login(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	Callback(w http.ResponseWriter, r *http.Request)
}

type GoogleOauth struct {
	Logger    *slog.Logger
	Config    *oauth2.Config
	Store     *sessions.CookieStore
	UserStore *store.PostgresUserStore
}

func NewGoogleOauth(logger *slog.Logger, store *sessions.CookieStore, userStore *store.PostgresUserStore) (*GoogleOauth, error) {

	return &GoogleOauth{
		Logger: logger,
		Config: &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID_USER"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET_USER"),
			RedirectURL:  fmt.Sprintf("%s/auth/google/callback", os.Getenv("NEXT_PUBLIC_BACKEND_URL")),
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.profile", "https://www.googleapis.com/auth/userinfo.email"},
			Endpoint:     google.Endpoint,
		},
		Store:     store,
		UserStore: userStore,
	}, nil
}

func (g *GoogleOauth) Login(w http.ResponseWriter, r *http.Request) {
	url, err := BeginOAuth(w, g.Config, UserStateCookie)
	if err != nil {
		g.Logger.Error("Error starting user oauth flow", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (g *GoogleOauth) Logout(w http.ResponseWriter, r *http.Request) {
	session, _ := g.Store.Get(r, "nazrein_session")

	for key := range session.Values {
		delete(session.Values, key)
	}

	session.Options.MaxAge = -1

	err := session.Save(r, w)
	if err != nil {
		g.Logger.Warn("Error clearing session", "err", err)
	}

	redirectURL := os.Getenv("FRONTEND_URL")
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (g *GoogleOauth) Callback(w http.ResponseWriter, r *http.Request) {
	verifier, err := CompleteOAuth(w, r, UserStateCookie)
	if err != nil {
		g.Logger.Warn("Rejecting user oauth callback", "err", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"Error": "Invalid login attempt"})
		return
	}

	code := r.URL.Query().Get("code")
	token, err := g.Config.Exchange(context.Background(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		g.Logger.Error("Error exchanging user token", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}

	client := g.Config.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		g.Logger.Error("Error getting user info", "err", err)
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
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Interal Server Error"})
	}

	var userID string
	var userRole string
	user, err := g.UserStore.GetUserByGoogleID(userInfo.GoogleID)

	if user == nil || err == sql.ErrNoRows {
		newUser := models.User{
			GoogleID: userInfo.GoogleID,
			Name:     userInfo.Name,
			Email:    userInfo.Email,
			ImageSrc: userInfo.Image,
			Role:     "USER",
		}

		err = g.UserStore.CreateUser(&newUser)
		if err != nil {
			g.Logger.Error("Error creating user", "err", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
			return
		}

		userID = newUser.ID.String()
		userRole = "USER"
	} else {
		userID = user.ID.String()
		userRole = user.Role
	}

	if err != nil && err != sql.ErrNoRows {
		g.Logger.Error("Error getting user by google id", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}

	session, _ := g.Store.Get(r, "nazrein_session")
	session.Values["user_id"] = userID
	session.Values["user_email"] = userInfo.Email
	session.Values["user_image"] = userInfo.Image
	session.Values["user_name"] = userInfo.Name
	session.Values["user_role"] = userRole

	err = session.Save(r, w)
	if err != nil {
		g.Logger.Error("Error saving session", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}

	redirectURL := os.Getenv("FRONTEND_URL") + "/dashboard"
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (g *GoogleOauth) AuthUser(w http.ResponseWriter, r *http.Request) {
	user, err := g.Store.Get(r, "nazrein_session")
	if err != nil {
		g.Logger.Warn("Error getting session", "err", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "Not Authenticated"})
		return
	}

	userEmail, emailOk := user.Values["user_email"].(string)
	userIDStr, idOk := user.Values["user_id"].(string)
	userName, nameOk := user.Values["user_name"].(string)
	userImage, imageOk := user.Values["user_image"].(string)
	userRole, roleOk := user.Values["user_role"].(string)

	if !emailOk || !idOk || !nameOk || !imageOk || !roleOk || userEmail == "" || userIDStr == "" || userName == "" || userImage == "" || userRole == "" {
		g.Logger.Warn("Invalid or missing user data in session")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "Not Authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		g.Logger.Warn("Invalid user ID format in session", "err", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "Not Authenticated"})
		return
	}

	userInfo := map[string]interface{}{
		"id":    userID,
		"email": userEmail,
		"name":  userName,
		"image": userImage,
		"role":  userRole,
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": userInfo})
}
