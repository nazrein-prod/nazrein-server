package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/grvbrk/nazrein_server/internal/auth"
	"github.com/grvbrk/nazrein_server/internal/middlewares"
	"github.com/grvbrk/nazrein_server/internal/store"
	"github.com/grvbrk/nazrein_server/internal/utils"
)

// var ctx = context.Background()

type VideoHandler struct {
	VideoStore store.VideoStore
	Logger     *log.Logger
	Oauth      *auth.GoogleOauth
}

func NewVideoHandler(videoStore store.VideoStore, logger *log.Logger, oauth *auth.GoogleOauth) *VideoHandler {
	return &VideoHandler{
		VideoStore: videoStore,
		Logger:     logger,
		Oauth:      oauth,
	}
}

func (vh *VideoHandler) HandlerGetVideos(w http.ResponseWriter, r *http.Request) {
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		vh.Logger.Println("Error: page parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		vh.Logger.Println("Error: limit parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	sortByStr := r.URL.Query().Get("sortBy")
	if sortByStr == "" {
		vh.Logger.Println("Error: sortBy parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	query := r.URL.Query().Get("q")

	searchTypeStr := r.URL.Query().Get("type")
	if searchTypeStr == "" {
		vh.Logger.Println("Error: searchType parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		vh.Logger.Printf("Error: invalid page parameter '%s': %v", pageStr, err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}
	if page < 1 {
		vh.Logger.Printf("Error: page parameter must be >= 1, got %d", page)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		vh.Logger.Printf("Error: invalid limit parameter '%s': %v", limitStr, err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}
	if limit < 1 || limit > 100 {
		vh.Logger.Printf("Error: limit parameter must be between 1 and 100, got %d", limit)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	sortBy := store.ValidateSortBy(sortByStr)
	if string(sortBy) != sortByStr {
		vh.Logger.Printf("Warning: invalid sort_by parameter '%s', defaulting to 'popular'", sortByStr)
	}

	searchType := store.ValidateSearchType(searchTypeStr)
	if string(searchType) != searchTypeStr {
		vh.Logger.Printf("Warning: invalid type parameter '%s', defaulting to 'video'", searchTypeStr)
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
			vh.Logger.Printf("Error getting videos from store: %v", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
			return
		}

		utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": response})
		return
	}

	// Authenticated user - return videos with bookmark information
	response, err := vh.VideoStore.GetVideosWithUserBookmarks(params, user.ID)
	if err != nil {
		vh.Logger.Printf("Error getting videos with bookmarks from store: %v", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": response})
}

func (vh *VideoHandler) HandlerGetVideosByUserID(w http.ResponseWriter, r *http.Request) {

	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		vh.Logger.Println("No user found in context.")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	videos, err := vh.VideoStore.GetVideosByUserID(user.ID)
	if err != nil {
		vh.Logger.Println("Error getting videos from store", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": videos})

}

func (vh *VideoHandler) HandlerGetVideoByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		vh.Logger.Println("Error: id parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	videoID, err := uuid.Parse(id)
	if err != nil {
		vh.Logger.Println("Error parsing video id", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	video, err := vh.VideoStore.GetVideoByID(videoID)
	if err != nil {
		vh.Logger.Println("Error getting video from store", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": video})
}

func (vh *VideoHandler) HandlerGetBookmarkedVideosByUserID(w http.ResponseWriter, r *http.Request) {

	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		vh.Logger.Println("No user found in context.")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	bookmarkedVideos, err := vh.VideoStore.GetBookmarkedVideosByUserID(user.ID)
	if err != nil {
		vh.Logger.Println("Error getting bookmarked videos from store", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": bookmarkedVideos})
}

func (vh *VideoHandler) HandlerGetSimilarVideosByName(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" || len(query) < 2 {
		vh.Logger.Println("Error: query is either missing or too short")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	videos, err := vh.VideoStore.GetSimilarVideosByName(query)
	if err != nil {
		vh.Logger.Println("Error getting similar videos from store", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}
	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": videos})

}
