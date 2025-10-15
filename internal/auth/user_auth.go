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
	Logger    *log.Logger
	Config    *oauth2.Config
	Store     *sessions.CookieStore
	UserStore *store.PostgresUserStore
}

func NewGoogleOauth(logger *log.Logger, store *sessions.CookieStore, userStore *store.PostgresUserStore) (*GoogleOauth, error) {

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
	url := g.Config.AuthCodeURL("random-state-string", oauth2.AccessTypeOffline)
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
		g.Logger.Println("Error clearing session", err)
	}

	redirectURL := os.Getenv("FRONTEND_URL")
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (g *GoogleOauth) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	token, err := g.Config.Exchange(context.Background(), code)
	if err != nil {
		g.Logger.Println("Error exchanging user token", err)
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
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Interal Server Error"})
	}

	var userID string
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
			g.Logger.Println("Error creating user", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
			return
		}

		userID = newUser.ID.String()
	} else {
		userID = user.ID.String()
	}

	if err != nil && err != sql.ErrNoRows {
		g.Logger.Println("Error getting user by google id", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}

	session, _ := g.Store.Get(r, "nazrein_session")
	session.Values["user_id"] = userID
	session.Values["user_email"] = userInfo.Email
	session.Values["user_image"] = userInfo.Image
	session.Values["user_name"] = userInfo.Name
	// session.Options.Path = "/"
	// session.Options.MaxAge = 0
	// session.Options.Secure = false
	// session.Options.SameSite = http.SameSiteLaxMode

	err = session.Save(r, w)
	if err != nil {
		g.Logger.Println("Error saving session", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}

	redirectURL := os.Getenv("FRONTEND_URL") + "/dashboard"
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (g *GoogleOauth) AuthUser(w http.ResponseWriter, r *http.Request) {
	user, err := g.Store.Get(r, "nazrein_session")
	if err != nil {
		g.Logger.Println("Error getting session", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "Not Authenticated"})
		return
	}

	userEmail, emailOk := user.Values["user_email"].(string)
	userIDStr, idOk := user.Values["user_id"].(string)
	userName, nameOk := user.Values["user_name"].(string)
	userImage, imageOk := user.Values["user_image"].(string)

	if !emailOk || !idOk || !nameOk || !imageOk || userEmail == "" || userIDStr == "" || userName == "" || userImage == "" {
		g.Logger.Println("Invalid or missing user data in session")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "Not Authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		g.Logger.Println("Invalid user ID format in session:", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"error": "Not Authenticated"})
		return
	}

	userInfo := map[string]interface{}{
		"id":    userID,
		"email": userEmail,
		"name":  userName,
		"image": userImage,
		"role":  "USER",
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": userInfo})

}
