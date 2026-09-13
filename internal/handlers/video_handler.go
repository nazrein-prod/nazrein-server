package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/grvbrk/nazrein_server/internal/auth"
	"github.com/grvbrk/nazrein_server/internal/middlewares"
	"github.com/grvbrk/nazrein_server/internal/store"
	"github.com/grvbrk/nazrein_server/internal/utils"
)

type VideoHandler struct {
	VideoStore store.VideoStore
	Logger     *slog.Logger
	Oauth      *auth.GoogleOauth
}

func NewVideoHandler(videoStore store.VideoStore, logger *slog.Logger, oauth *auth.GoogleOauth) *VideoHandler {
	return &VideoHandler{
		VideoStore: videoStore,
		Logger:     logger,
		Oauth:      oauth,
	}
}

// HandlerGetVideos lists public videos. Query parameters are bound and
// validated from the OpenAPI spec before this is called; params arrives already
// parsed and defaulted.
func (vh *VideoHandler) HandlerGetVideos(w http.ResponseWriter, r *http.Request, params store.GetVideosParams) {
	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		// No authenticated user - return videos without bookmark information
		response, err := vh.VideoStore.GetVideos(params)
		if err != nil {
			vh.Logger.Error("Error getting videos from store", "err", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
			return
		}

		utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": response})
		return
	}

	// Authenticated user - return videos with bookmark information
	response, err := vh.VideoStore.GetVideosWithUserBookmarks(params, user.ID)
	if err != nil {
		vh.Logger.Error("Error getting videos with bookmarks from store", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": response})
}

func (vh *VideoHandler) HandlerGetVideosByUserID(w http.ResponseWriter, r *http.Request) {

	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		vh.Logger.Warn("No user found in context")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	videos, err := vh.VideoStore.GetVideosByUserID(user.ID)
	if err != nil {
		vh.Logger.Error("Error getting videos from store", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": videos})

}

func (vh *VideoHandler) HandlerGetVideoByID(w http.ResponseWriter, r *http.Request, videoID uuid.UUID) {
	video, err := vh.VideoStore.GetVideoByID(videoID)
	if err != nil {
		vh.Logger.Error("Error getting video from store", "video_id", videoID, "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := vh.VideoStore.RecordVisit(ctx, videoID); err != nil {
			vh.Logger.Warn("Error recording video visit", "video_id", videoID, "err", err)
		}
	}()

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": video})
}

func (vh *VideoHandler) HandlerGetBookmarkedVideosByUserID(w http.ResponseWriter, r *http.Request) {

	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		vh.Logger.Warn("No user found in context")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	bookmarkedVideos, err := vh.VideoStore.GetBookmarkedVideosByUserID(user.ID)
	if err != nil {
		vh.Logger.Error("Error getting bookmarked videos from store", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": bookmarkedVideos})
}

func (vh *VideoHandler) HandlerGetSimilarVideosByName(w http.ResponseWriter, r *http.Request, query string) {
	videos, err := vh.VideoStore.GetSimilarVideosByName(query)
	if err != nil {
		vh.Logger.Error("Error getting similar videos from store", "query", query, "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": videos})

}
