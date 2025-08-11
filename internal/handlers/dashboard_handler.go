package handlers

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/grvbrk/nazrein_server/internal/store"
	"github.com/grvbrk/nazrein_server/internal/utils"
)

type DashboardHandler struct {
	dashboardStore store.DashboardStore
	logger         *log.Logger
}

func NewDashboardHandler(dashboardStore store.DashboardStore, logger *log.Logger) *DashboardHandler {
	return &DashboardHandler{
		dashboardStore: dashboardStore,
		logger:         logger,
	}
}

func (dh *DashboardHandler) HandlerGetDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "user_id")
	if id == "" {
		dh.logger.Println("error getting user id from url param")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad request"})
	}

	userID, err := uuid.Parse(id)
	if err != nil {
		dh.logger.Println("error parsing user id")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad request"})
	}

	dashboard, err := dh.dashboardStore.GetDashboardMetricsByUserID(userID)
	if err != nil {
		dh.logger.Println("error getting dashboard metrics")
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal server error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": dashboard})
}
