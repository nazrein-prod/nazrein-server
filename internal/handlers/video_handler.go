package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
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

func (vh *VideoHandler) HandlerGetVideos(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		vh.Logger.Warn("page parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		vh.Logger.Warn("limit parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	sortByStr := r.URL.Query().Get("sortBy")
	if sortByStr == "" {
		vh.Logger.Warn("sortBy parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	query := r.URL.Query().Get("q")

	searchTypeStr := r.URL.Query().Get("type")
	if searchTypeStr == "" {
		vh.Logger.Warn("searchType parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		vh.Logger.Warn("invalid page parameter", "page", pageStr, "err", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}
	if page < 1 {
		vh.Logger.Warn("page parameter must be >= 1", "page", page)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		vh.Logger.Warn("invalid limit parameter", "limit", limitStr, "err", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}
	if limit < 1 || limit > 100 {
		vh.Logger.Warn("limit parameter must be between 1 and 100", "limit", limit)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	sortBy := store.ValidateSortBy(sortByStr)
	if string(sortBy) != sortByStr {
		vh.Logger.Warn("invalid sort_by parameter, defaulting to popular", "sortBy", sortByStr)
	}

	searchType := store.ValidateSearchType(searchTypeStr)
	if string(searchType) != searchTypeStr {
		vh.Logger.Warn("invalid type parameter, defaulting to video", "type", searchTypeStr)
	}

	params := store.GetVideosParams{
		Page:   page,
		Limit:  limit,
		SortBy: sortBy,
		Query:  query,
		Type:   searchType,
	}

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

func (vh *VideoHandler) HandlerGetVideoByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		vh.Logger.Warn("id parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	videoID, err := uuid.Parse(id)
	if err != nil {
		vh.Logger.Warn("Error parsing video id", "id", id, "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

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

func (vh *VideoHandler) HandlerGetSimilarVideosByName(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" || len(query) < 2 {
		vh.Logger.Warn("query is either missing or too short", "query", query)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	videos, err := vh.VideoStore.GetSimilarVideosByName(query)
	if err != nil {
		vh.Logger.Error("Error getting similar videos from store", "query", query, "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": videos})

}
