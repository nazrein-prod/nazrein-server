package handlers

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/grvbrk/nazrein_server/internal/auth"
	"github.com/grvbrk/nazrein_server/internal/middlewares"
	"github.com/grvbrk/nazrein_server/internal/store"
	"github.com/grvbrk/nazrein_server/internal/utils"
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

	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		bh.Logger.Println("No user found in context.")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	videoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		bh.Logger.Println("Error parsing video id", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	err = bh.BookmarkStore.CreateBookmark(videoID, user.ID)
	if err != nil {
		bh.Logger.Println("Error creating bookmark", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "Success"})
}

func (bh *BookmarkHandler) HandlerDeleteBookmark(w http.ResponseWriter, r *http.Request) {

	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		bh.Logger.Println("No user found in context.")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	videoID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		bh.Logger.Println("Error parsing video id", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	err = bh.BookmarkStore.DeleteBookmark(videoID, user.ID)
	if err != nil {
		bh.Logger.Println("Error deleting bookmark", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "Success"})
}
