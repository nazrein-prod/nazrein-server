package models

import (
	"time"

	"github.com/google/uuid"
)

type VideoRequest struct {
	Id              uuid.UUID  `json:"id"`
	Status          string     `json:"status"`
	Link            string     `json:"link"`
	Youtube_ID      string     `json:"youtube_id"`
	UserId          string     `json:"user_id"`
	ProcessedBy     *uuid.UUID `json:"processed_by"`
	ProcessedAt     *time.Time `json:"processed_at"`
	RejectionReason *string    `json:"rejection_reason"`
	Created_At      time.Time  `json:"created_at"`
	Updated_At      time.Time  `json:"updated_at"`
}
