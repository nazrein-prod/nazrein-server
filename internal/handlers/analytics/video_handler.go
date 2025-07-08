package analytics

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/grvbrk/track-yt-video/internal/store/analytics"
	"github.com/grvbrk/track-yt-video/internal/utils"
)

type AnalyticsVideoHandler struct {
	AnalyticsVideoStore analytics.AnalyticsVideoStore
	Logger              *log.Logger
}

func NewAnalyticsVideoHandler(analyticsVideoStore analytics.AnalyticsVideoStore, logger *log.Logger) *AnalyticsVideoHandler {
	return &AnalyticsVideoHandler{
		AnalyticsVideoStore: analyticsVideoStore,
		Logger:              logger,
	}
}

func (ah *AnalyticsVideoHandler) HandlerGetVideoAnalyticsByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ah.Logger.Println("Error: id parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	response, err := ah.AnalyticsVideoStore.GetVideoAnalyticsByID(id)
	if err != nil {
		ah.Logger.Println("Error getting video analytics from store", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": response})
}
