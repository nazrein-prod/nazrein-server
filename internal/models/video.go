package models

import (
	"time"

	"github.com/google/uuid"
)

type Video struct {
	Id            uuid.UUID `json:"id"`
	Link          string    `json:"link"`
	Published_At  time.Time `json:"published_at"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Thumbnail     string    `json:"thumbnail"`
	Youtube_ID    string    `json:"youtube_id"`
	Channel_Title string    `json:"channel_title"`
	Channel_ID    string    `json:"channel_id"`
	User_ID       uuid.UUID `json:"user_id"`
	Is_Active     bool      `json:"is_active"`
	Visits        int       `json:"visits"`
	Created_At    time.Time `json:"created_at"`
	Updated_At    time.Time `json:"updated_at"`
}
