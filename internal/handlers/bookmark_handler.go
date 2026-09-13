package handlers

import (
	"log/slog"
	"net/http"

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
	Logger        *slog.Logger
}

func NewBookmarkHandler(videoStore store.VideoStore, bookmarkStore store.BookmarkStore, userStore store.UserStore, oauth *auth.GoogleOauth, logger *slog.Logger) *BookmarkHandler {
	return &BookmarkHandler{
		VideoStore:    videoStore,
		BookmarkStore: bookmarkStore,
		UserStore:     userStore,
		Oauth:         oauth,
		Logger:        logger,
	}
}

func (bh *BookmarkHandler) HandlerCreateBookmark(w http.ResponseWriter, r *http.Request, videoID uuid.UUID) {

	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		bh.Logger.Warn("No user found in context")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	err := bh.BookmarkStore.CreateBookmark(videoID, user.ID)
	if err != nil {
		bh.Logger.Error("Error creating bookmark", "video_id", videoID, "user_id", user.ID, "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "Success"})
}

func (bh *BookmarkHandler) HandlerDeleteBookmark(w http.ResponseWriter, r *http.Request, videoID uuid.UUID) {

	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		bh.Logger.Warn("No user found in context")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	err := bh.BookmarkStore.DeleteBookmark(videoID, user.ID)
	if err != nil {
		bh.Logger.Error("Error deleting bookmark", "video_id", videoID, "user_id", user.ID, "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "Success"})
}
