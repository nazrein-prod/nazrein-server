package handlers

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/grvbrk/track-yt-video/internal/auth"
	"github.com/grvbrk/track-yt-video/internal/store"
	"github.com/grvbrk/track-yt-video/internal/utils"
)

type BookmarkHandler struct {
	VideoStore    store.VideoStore
	BookmarkStore store.BookmarkStore
	UserStore     store.UserStore
	Oauth         *auth.GoogleOauth
	Logger        *log.Logger
}

func NewBookmarkHandler(videoStore store.VideoStore, bookmarkStore store.BookmarkStore, userStore store.UserStore, oauth *auth.GoogleOauth, logger *log.Logger) *BookmarkHandler {
	return &BookmarkHandler{
		VideoStore:    videoStore,
		BookmarkStore: bookmarkStore,
		UserStore:     userStore,
		Oauth:         oauth,
		Logger:        logger,
	}
}

func (bh *BookmarkHandler) HandlerCreateBookmark(w http.ResponseWriter, r *http.Request) {

	currentUser, err := bh.Oauth.GetUser(r)
	if err != nil {
		bh.Logger.Println("Error fetching user", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	if currentUser == nil {
		bh.Logger.Println("No user found", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	videoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		bh.Logger.Println("Error parsing video id", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	err = bh.BookmarkStore.CreateBookmark(videoID, currentUser.ID)
	if err != nil {
		bh.Logger.Println("Error creating bookmark", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "Success"})
}

func (bh *BookmarkHandler) HandlerDeleteBookmark(w http.ResponseWriter, r *http.Request) {
	currentUser, err := bh.Oauth.GetUser(r)
	if err != nil {
		bh.Logger.Println("Error fetching user", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	if currentUser == nil {
		bh.Logger.Println("No user found", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	videoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		bh.Logger.Println("Error parsing video id", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	err = bh.BookmarkStore.DeleteBookmark(videoID, currentUser.ID)
	if err != nil {
		bh.Logger.Println("Error deleting bookmark", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "Success"})
}
