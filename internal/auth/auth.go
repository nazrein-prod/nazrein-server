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
	"github.com/joho/godotenv"
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
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load env variables: %w", err)
	}
	return &GoogleOauth{
		Logger: logger,
		Config: &oauth2.Config{
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			RedirectURL:  "http://localhost:8080/auth/google/callback", // FIX
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
	session, _ := g.Store.Get(r, "session")
	delete(session.Values, "user_email")
	session.Save(r, w)

	http.Redirect(w, r, "http://localhost:3000", http.StatusSeeOther)
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

	if err != nil {
		g.Logger.Println("Error getting user by google id", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"Error": "Internal Server Error"})
		return
	}

	session, _ := g.Store.Get(r, "session")
	session.Values["user_id"] = userID
	session.Values["user_email"] = userInfo.Email
	session.Values["user_image"] = userInfo.Image
	session.Values["user_name"] = userInfo.Name
	session.Options.Path = "/"
	session.Options.MaxAge = 0
	// session.Options.Secure = false
	// session.Options.SameSite = http.SameSiteLaxMode

	err = session.Save(r, w)
	if err != nil {
		g.Logger.Println("Error saving session", err)
	}

	http.Redirect(w, r, "http://localhost:3000/dashboard", http.StatusSeeOther)
}

func (g *GoogleOauth) AuthUser(w http.ResponseWriter, r *http.Request) {

	session, err := g.Store.Get(r, "session")
	if err != nil {
		g.Logger.Println("Failed to decode session:", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"Error": "Unauthorized"})
		return
	}

	email, ok := session.Values["user_email"].(string)
	if !ok || email == "" {
		g.Logger.Println("No user email found in session", email, ok)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"Error": "Unauthorized"})
		return
	}

	userId, _ := session.Values["user_id"].(string)
	userImage, _ := session.Values["user_image"].(string)
	userName, _ := session.Values["user_name"].(string)
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{
		"user_id":    userId,
		"user_email": email,
		"user_image": userImage,
		"user_name":  userName,
	})
}

func (g *GoogleOauth) GetUser(r *http.Request) (*models.User, error) {
	session, err := g.Store.Get(r, "session")

	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	userEmail, ok := session.Values["user_email"].(string)
	if !ok || userEmail == "" {
		return nil, fmt.Errorf("no user email found in session")
	}

	id, ok := session.Values["user_id"].(string)
	if !ok || id == "" {
		return nil, fmt.Errorf("no user id found in session")
	}

	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user id: %w", err)
	}

	return &models.User{
		ID:    userID,
		Email: userEmail,
	}, nil
}
