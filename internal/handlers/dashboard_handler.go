package handlers

import (
	"log/slog"
	"net/http"

	"github.com/grvbrk/nazrein_server/internal/middlewares"
	"github.com/grvbrk/nazrein_server/internal/store"
	"github.com/grvbrk/nazrein_server/internal/utils"
)

type DashboardHandler struct {
	dashboardStore store.DashboardStore
	Logger         *slog.Logger
}

func NewDashboardHandler(dashboardStore store.DashboardStore, logger *slog.Logger) *DashboardHandler {
	return &DashboardHandler{
		dashboardStore: dashboardStore,
		Logger:         logger,
	}
}

func (dh *DashboardHandler) HandlerGetDashboardMetrics(w http.ResponseWriter, r *http.Request) {

	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		dh.Logger.Warn("No user found in context")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	dashboard, err := dh.dashboardStore.GetDashboardMetricsByUserID(user.ID)
	if err != nil {
		dh.Logger.Error("Error getting dashboard metrics", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal server error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"data": dashboard})
}
