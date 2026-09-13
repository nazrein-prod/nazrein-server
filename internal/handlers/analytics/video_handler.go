package analytics

import (
	"log/slog"
	"net/http"

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

// HandlerGetVideoAnalyticsByID returns one page of a video's snapshot timeline.
// id, limit and offset are bound from the OpenAPI spec; the store clamps limit
// and floors offset.
func (ah *AnalyticsVideoHandler) HandlerGetVideoAnalyticsByID(w http.ResponseWriter, r *http.Request, id string, limit, offset int) {
	response, err := ah.AnalyticsVideoStore.GetVideoAnalyticsByID(id, limit, offset)
	if err != nil {
		ah.Logger.Error("Error getting video analytics from store", "id", id, "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": response})
}
