package analytics

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/grvbrk/nazrein_server/internal/store/analytics"
	"github.com/grvbrk/nazrein_server/internal/utils"
)

type AnalyticsVideoHandler struct {
	AnalyticsVideoStore analytics.AnalyticsVideoStore
	Logger              *slog.Logger
}

func NewAnalyticsVideoHandler(analyticsVideoStore analytics.AnalyticsVideoStore, logger *slog.Logger) *AnalyticsVideoHandler {
	return &AnalyticsVideoHandler{
		AnalyticsVideoStore: analyticsVideoStore,
		Logger:              logger,
	}
}

func (ah *AnalyticsVideoHandler) HandlerGetVideoAnalyticsByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ah.Logger.Warn("id parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	// Both optional; the store clamps limit and floors offset.
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	response, err := ah.AnalyticsVideoStore.GetVideoAnalyticsByID(id, limit, offset)
	if err != nil {
		ah.Logger.Error("Error getting video analytics from store", "id", id, "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": response})
}
