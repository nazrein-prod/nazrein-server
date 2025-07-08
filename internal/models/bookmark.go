package models

import (
	"time"

	"github.com/google/uuid"
)

type Bookmark struct {
	Id         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	VideoID    uuid.UUID `json:"video_id"`
	Created_At time.Time `json:"created_at"`
	Updated_At time.Time `json:"updated_at"`
}
