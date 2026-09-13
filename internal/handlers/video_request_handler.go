package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/grvbrk/nazrein_server/internal/auth"
	"github.com/grvbrk/nazrein_server/internal/middlewares"
	"github.com/grvbrk/nazrein_server/internal/models"
	"github.com/grvbrk/nazrein_server/internal/store"
	"github.com/grvbrk/nazrein_server/internal/utils"
)

type VideoRequestHandler struct {
	VideoRequestStore store.VideoRequestStore
	Logger            *slog.Logger
	Oauth             *auth.GoogleOauth
}

func NewVideoRequestHandler(videoReqStore store.VideoRequestStore, logger *slog.Logger, oauth *auth.GoogleOauth) *VideoRequestHandler {
	return &VideoRequestHandler{
		VideoRequestStore: videoReqStore,
		Logger:            logger,
		Oauth:             oauth,
	}
}

func (vrh *VideoRequestHandler) HandlerCreateVideoRequest(w http.ResponseWriter, r *http.Request) {
	var req models.VideoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		vrh.Logger.Warn("Error decoding request body in handler", "err", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		vrh.Logger.Warn("No user found in context")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	var totalRequests int
	if user.Role == "USER" {
		totalRequests, err = vrh.VideoRequestStore.GetTotalPendingRequestsByUserID(user.ID)
		if err != nil {
			vrh.Logger.Error("Error getting total requests by user id", "err", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
			return
		}

		if totalRequests >= 3 {
			vrh.Logger.Warn("User has already made 3 video requests", "user_id", user.ID)
			utils.WriteJSON(w, http.StatusForbidden, utils.Envelope{"message": "You have already made 3 video requests"})
			return
		}
	}

	err = vrh.VideoRequestStore.CreateVideoRequest(&req, user.ID)
	if err != nil {
		vrh.Logger.Error("Error creating video request in store", "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "Success"})
}

func (vrh *VideoRequestHandler) HandlerDeleteVideoRequestByID(w http.ResponseWriter, r *http.Request) {
	videoRequestID := chi.URLParam(r, "id")
	if videoRequestID == "" {
		vrh.Logger.Warn("No video request id found in url")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		vrh.Logger.Warn("No user found in context")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	requestID, err := uuid.Parse(videoRequestID)
	if err != nil {
		vrh.Logger.Warn("Parsing error from string to uuid", "id", videoRequestID, "err", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	userID, err := vrh.VideoRequestStore.GetVideoRequestUserID(requestID)
	if err != nil {
		vrh.Logger.Warn("Error getting video request user id", "request_id", requestID, "err", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	if user.ID != userID {
		vrh.Logger.Warn("user id does not match request owner", "user_id", user.ID, "owner_id", userID)
		utils.WriteJSON(w, http.StatusForbidden, utils.Envelope{"error": "Forbidden"})
		return
	}

	err = vrh.VideoRequestStore.DeleteVideoRequest(requestID)
	if err != nil {
		vrh.Logger.Error("Error deleting video request by id in handler", "request_id", requestID, "err", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "Sucess"})

}

func (vrh *VideoRequestHandler) HandlerGetAllVideoRequestsByUserID(w http.ResponseWriter, r *http.Request) {

	user, ok := middlewares.GetUserFromContext(r)
	if !ok {
		vrh.Logger.Warn("No user found in context")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	videoRequestArr, err := vrh.VideoRequestStore.GetAllVideoRequestByUserID(user.ID)
	if err != nil {
		vrh.Logger.Error("Error getting video requests by user id", "user_id", user.ID, "err", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"data": videoRequestArr})
}
