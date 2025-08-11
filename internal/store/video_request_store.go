package store

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/grvbrk/nazrein_server/internal/models"
)

type PostgresVideoRequestStore struct {
	db *sql.DB
}

func NewPostgresVideoRequestStore(db *sql.DB) *PostgresVideoRequestStore {
	return &PostgresVideoRequestStore{db: db}
}

type VideoRequestStore interface {
	CreateVideoRequest(vr *models.VideoRequest, userID uuid.UUID) error
	DeleteVideoRequest(requestID uuid.UUID) error
	GetAllVideoRequestByUserID(UserID uuid.UUID) ([]models.VideoRequest, error)
	GetVideoRequestUserID(requestID uuid.UUID) (uuid.UUID, error)
	GetTotalPendingRequestsByUserID(UserID uuid.UUID) (int, error)
}

func (pvr *PostgresVideoRequestStore) CreateVideoRequest(vr *models.VideoRequest, userID uuid.UUID) error {

	query := `
	INSERT INTO video_requests (link, youtube_id, user_id)
	VALUES ($1, $2, $3)
	`

	_, err := pvr.db.Exec(query, vr.Link, vr.Youtube_ID, userID)
	if err != nil {
		return fmt.Errorf("failed to insert video request: %w", err)
	}
	return nil
}

func (pvr *PostgresVideoRequestStore) DeleteVideoRequest(requestID uuid.UUID) error {

	query := `
	DELETE FROM video_requests
	WHERE id = $1
	`

	err := pvr.db.QueryRow(query, requestID).Err()
	if err != nil {
		return fmt.Errorf("failed to delete video request: %w", err)
	}

	return nil
}

func (pvr *PostgresVideoRequestStore) GetAllVideoRequestByUserID(UserID uuid.UUID) ([]models.VideoRequest, error) {
	var videoRequests []models.VideoRequest
	query := `
	SELECT * FROM video_requests
	WHERE user_id = $1
	`

	rows, err := pvr.db.Query(query, UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to select video requests: %w", err)
	}

	for rows.Next() {
		var videoRequest models.VideoRequest
		err = rows.Scan(&videoRequest.Id, &videoRequest.Status, &videoRequest.Link, &videoRequest.Youtube_ID, &videoRequest.UserId, &videoRequest.ProcessedBy, &videoRequest.ProcessedAt, &videoRequest.RejectionReason, &videoRequest.Created_At, &videoRequest.Updated_At)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video request: %w", err)
		}
		videoRequests = append(videoRequests, videoRequest)
	}
	return videoRequests, nil
}

func (pvr *PostgresVideoRequestStore) GetVideoRequestUserID(requestID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID

	query := `
		SELECT user_id
		FROM video_requests
		WHERE id = $1
		`

	err := pvr.db.QueryRow(query, requestID).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to select user id of video request: %w", err)
	}

	return userID, nil
}

func (pvr *PostgresVideoRequestStore) GetTotalPendingRequestsByUserID(UserID uuid.UUID) (int, error) {

	var totalRequests int

	query := `
		SELECT COUNT(*)
		FROM video_requests
		WHERE user_id = $1 AND status = 'PENDING'
	`

	err := pvr.db.QueryRow(query, UserID).Scan(&totalRequests)
	if err != nil {
		return 0, fmt.Errorf("failed to select total requests of user: %w", err)
	}

	return totalRequests, nil

}
