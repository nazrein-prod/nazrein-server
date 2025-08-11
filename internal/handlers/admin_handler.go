package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/grvbrk/nazrein_server/internal/auth"
	"github.com/grvbrk/nazrein_server/internal/models"
	"github.com/grvbrk/nazrein_server/internal/store/admin"
	"github.com/grvbrk/nazrein_server/internal/utils"
)

type YouTubeResponse struct {
	Items []struct {
		Snippet struct {
			PublishedAt time.Time `json:"publishedAt"`
			ChannelId   string    `json:"channelId"`
			Title       string    `json:"title"`
			Description string    `json:"description"`
			Thumbnails  struct {
				Default struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"default"`
				Medium struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"medium"`
				High struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"high"`
				Standard struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"standard"`
				Maxres struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"maxres"`
			} `json:"thumbnails"`
			ChannelTitle string `json:"channelTitle"`
		} `json:"snippet"`
	} `json:"items"`
}

type AdminHandler struct {
	AdminVideoStore        admin.AdminVideoStore
	AdminUserStore         admin.AdminUserStore
	AdminVideoRequestStore admin.AdminVideoRequestStore
	Logger                 *log.Logger
	Oauth                  *auth.AdminGoogleOauth
}

func NewAdminHandler(adminVideoStore admin.AdminVideoStore, adminUserStore admin.AdminUserStore, adminVideoRequestStore admin.AdminVideoRequestStore, logger *log.Logger, oauth *auth.AdminGoogleOauth) *AdminHandler {
	return &AdminHandler{
		AdminVideoStore:        adminVideoStore,
		AdminUserStore:         adminUserStore,
		AdminVideoRequestStore: adminVideoRequestStore,
		Logger:                 logger,
		Oauth:                  oauth,
	}
}

func (ah *AdminHandler) HandlerGetVideoRequests(w http.ResponseWriter, r *http.Request) {
	responseArr, err := ah.AdminVideoStore.GetAllVideoRequest()
	if err != nil {
		ah.Logger.Println("Error fetching all video requests", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"data": responseArr})

}

func (ah *AdminHandler) HandlerApproveVideoRequest(w http.ResponseWriter, r *http.Request) {

	type Request struct {
		UserID    string `json:"user_id"`
		RequestID string `json:"request_id"`
		Link      string `json:"link"`
		YoutubeID string `json:"youtube_id"`
	}

	var req Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ah.Logger.Println("Error decoding request body:", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		ah.Logger.Println("Error parsing user id", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	user, err := ah.AdminUserStore.GetUserByID(userID)
	if err != nil {
		ah.Logger.Println("Error fetching user", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Bad Request"})
		return
	}

	if user.Role == "USER" && user.Videos_Tracked >= 3 {
		ah.Logger.Println("User has reached track limit")
		utils.WriteJSON(w, http.StatusForbidden, utils.Envelope{"message": "User has reached track limit"})
		return
	}

	apiKey := os.Getenv("YOUTUBE_API_KEY")
	if apiKey == "" {
		ah.Logger.Println("Error: YOUTUBE_API_KEY environment variable is not set")
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	url := fmt.Sprintf(
		"https://youtube.googleapis.com/youtube/v3/videos?part=snippet&id=%s&key=%s",
		req.YoutubeID, apiKey,
	)

	resp, err := http.Get(url)
	if err != nil {
		ah.Logger.Println("Error fetching video from youtube v3 api", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ah.Logger.Printf("Non-OK response: %d %s\n", resp.StatusCode, resp.Status)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ah.Logger.Printf("Failed to read response: %v\n", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	var ytResp YouTubeResponse
	if err := json.Unmarshal(body, &ytResp); err != nil {
		ah.Logger.Printf("Failed to decode JSON: %v\n", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	if len(ytResp.Items) == 0 {
		ah.Logger.Println("No video found.")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	video := models.Video{
		Link:          req.Link,
		Youtube_ID:    req.YoutubeID,
		Published_At:  ytResp.Items[0].Snippet.PublishedAt,
		Title:         ytResp.Items[0].Snippet.Title,
		Description:   ytResp.Items[0].Snippet.Description,
		Thumbnail:     ytResp.Items[0].Snippet.Thumbnails.High.URL,
		Channel_Title: ytResp.Items[0].Snippet.ChannelTitle,
		Channel_ID:    ytResp.Items[0].Snippet.ChannelId,
		User_ID:       userID,
		Is_Active:     true,
		Created_At:    time.Now(),
		Updated_At:    time.Now(),
	}

	user_uuid, err := uuid.Parse(req.UserID)
	if err != nil {
		ah.Logger.Println("Error parsing user id", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	requestID, err := uuid.Parse(req.RequestID)
	if err != nil {
		ah.Logger.Println("Error parsing request id", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	err = ah.AdminVideoStore.CreateVideo(&video, user_uuid, requestID)
	if err != nil {
		ah.Logger.Println("Error creating video in store:", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"message": "Success"})

}

func (ah *AdminHandler) HandlerUpdateVideoRequest(w http.ResponseWriter, r *http.Request) {
	type PatchRequest struct {
		UserID          string `json:"user_id"`
		Status          string `json:"status"`
		ProcessedBy     string `json:"processed_by"`
		RejectionReason string `json:"rejection_reason"`
	}

	rid := chi.URLParam(r, "request_id")
	if rid == "" {
		ah.Logger.Println("Error: request_id parameter is missing")
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	requestID, err := uuid.Parse(rid)
	if err != nil {
		ah.Logger.Println("Error parsing request id", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	var req PatchRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		ah.Logger.Println("Error decoding request body:", err)
		utils.WriteJSON(w, http.StatusBadRequest, utils.Envelope{"message": "Bad Request"})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		ah.Logger.Println("Error parsing user id", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	user, err := ah.AdminUserStore.GetUserByID(userID)
	if err != nil {
		ah.Logger.Println("Error fetching user", err)
		utils.WriteJSON(w, http.StatusUnauthorized, utils.Envelope{"message": "Not Authorized"})
		return
	}

	if user.ID != userID {
		ah.Logger.Println("User does not match")
		utils.WriteJSON(w, http.StatusForbidden, utils.Envelope{"message": "Forbidden"})
		return
	}

	err = ah.AdminVideoRequestStore.PatchVideoRequest(
		requestID,
		&req.Status,
		&req.ProcessedBy,
		&req.RejectionReason,
	)
	if err != nil {
		ah.Logger.Println("Error updating video request:", err)
		utils.WriteJSON(w, http.StatusInternalServerError, utils.Envelope{"message": "Internal Server Error"})
		return
	}

	utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "Success"})

}
