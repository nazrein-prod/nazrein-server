package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID `json:"id"`
	GoogleID       string    `json:"google_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	ImageSrc       string    `json:"image"`
	Role           string    `json:"role"`
	Videos_Tracked int       `json:"videos_tracked"`
	Created_At     time.Time `json:"created_at"`
	Updated_At     time.Time `json:"updated_at"`
}
