package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/grvbrk/track-yt-video/internal/auth"
	"github.com/grvbrk/track-yt-video/internal/models"
	"github.com/grvbrk/track-yt-video/internal/store"
	"github.com/grvbrk/track-yt-video/internal/utils"
)

type VideoRequestHandler struct {
	VideoRequestStore store.VideoRequestStore
	Logger            *log.Logger
	Oauth             *auth.GoogleOauth
}

func NewVideoRequestHandler(videoReqStore store.VideoRequestStore, logger *log.Logger, oauth *auth.GoogleOauth) *VideoRequestHandler {
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
		vrh.Logger.Println("Error decoding request body in handler", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	currentUser, err := vrh.Oauth.GetUser(r)
	if err != nil {
		vrh.Logger.Println("Error fetching user in oauth getuser", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	if currentUser == nil {
		vrh.Logger.Println("No user found in oauth getuser")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	var totalRequests int
	if currentUser.Role == "USER" {
		totalRequests, err = vrh.VideoRequestStore.GetTotalPendingRequestsByUserID(currentUser.ID)
		if err != nil {
			vrh.Logger.Println("Error getting total requests by user id", err)
			utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
			return
		}

		if totalRequests >= 3 {
			vrh.Logger.Println("User has already made 3 video requests")
			utils.WriteJSON(w, http.StatusForbidden, utils.Envelope{"message": "You have already made 3 video requests"})
			return
		}
	}

	err = vrh.VideoRequestStore.CreateVideoRequest(&req, currentUser.ID)
	if err != nil {
		vrh.Logger.Println("Error creating video request in store", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "Success"})
}

func (vrh *VideoRequestHandler) HandlerDeleteVideoRequestByID(w http.ResponseWriter, r *http.Request) {
	videoRequestID := chi.URLParam(r, "id")
	if videoRequestID == "" {
		vrh.Logger.Println("No video request id found in url")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	currentUser, err := vrh.Oauth.GetUser(r)
	if err != nil {
		vrh.Logger.Println("Error fetching user in oauth getuser", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	if currentUser == nil {
		vrh.Logger.Println("No user found in oauth getuser")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	requestID, err := uuid.Parse(videoRequestID)
	if err != nil {
		vrh.Logger.Println("Parsing error from string to uuid")
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	userID, err := vrh.VideoRequestStore.GetVideoRequestUserID(requestID)
	if err != nil {
		vrh.Logger.Println("Error getting video request user id", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	if currentUser.ID != userID {
		vrh.Logger.Println("user id does not match")
		utils.WriteJSON(w, http.StatusForbidden, utils.Envelope{"error": "Forbidden"})
		return
	}

	err = vrh.VideoRequestStore.DeleteVideoRequest(requestID)
	if err != nil {
		vrh.Logger.Println("Error deleting video request by id in handler", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "Sucess"})

}

func (vrh *VideoRequestHandler) HandlerGetAllVideoRequestsByUserID(w http.ResponseWriter, r *http.Request) {

	currentUser, err := vrh.Oauth.GetUser(r)
	if err != nil {
		vrh.Logger.Println("Error fetching user in oauth getuser", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	if currentUser == nil {
		vrh.Logger.Println("No user found in oauth getuser")
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	videoRequestArr, err := vrh.VideoRequestStore.GetAllVideoRequestByUserID(currentUser.ID)
	if err != nil {
		vrh.Logger.Println("Error getting video requests by user id", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"data": videoRequestArr})
}
