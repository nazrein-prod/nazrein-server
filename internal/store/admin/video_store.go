package admin

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grvbrk/nazrein_server/internal/models"
)

type AdminVideoRequest struct {
	Id         uuid.UUID   `json:"id"`
	Status     string      `json:"status"`
	Link       string      `json:"link"`
	Youtube_ID string      `json:"youtube_id"`
	User       models.User `json:"user"`
	Created_At time.Time   `json:"created_at"`
}

type AdminPostgresVideoStore struct {
	db *sql.DB
}

func NewPostgresAdminVideoStore(db *sql.DB) *AdminPostgresVideoStore {
	return &AdminPostgresVideoStore{db: db}
}

type AdminVideoStore interface {
	GetAllVideoRequest() ([]AdminVideoRequest, error)
	CreateVideo(video *models.Video, userId uuid.UUID, requestID uuid.UUID) error
}

func (a *AdminPostgresVideoStore) GetAllVideoRequest() ([]AdminVideoRequest, error) {

	results := []AdminVideoRequest{}

	query := `
		SELECT vr.id, vr.status, vr.link, vr.youtube_id, vr.created_at, u.id, u.name, u.image, u.videos_tracked
		FROM video_requests vr
		JOIN users u ON vr.user_id = u.id;
	`

	rows, err := a.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch all video request: %w", err)
	}

	defer rows.Close()

	for rows.Next() {
		var videoReq AdminVideoRequest
		var user models.User

		err := rows.Scan(
			&videoReq.Id,
			&videoReq.Status,
			&videoReq.Link,
			&videoReq.Youtube_ID,
			&videoReq.Created_At,
			&user.ID,
			&user.Name,
			&user.ImageSrc,
			&user.Videos_Tracked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		videoReq.User = user
		results = append(results, videoReq)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return results, nil

}

func (a *AdminPostgresVideoStore) CreateVideo(video *models.Video, userId uuid.UUID, requestID uuid.UUID) error {

	tx, err := a.db.Begin()
	if err != nil {
		fmt.Println("Error starting transaction", err)
		return err
	}

	defer func() {
		if rErr := tx.Rollback(); rErr != nil && rErr != sql.ErrTxDone {
			fmt.Printf("rollback error: %v", rErr)
		}
	}()

	query := `
	INSERT INTO videos (link, published_at, title, description, thumbnail, youtube_id, channel_title, channel_id, user_id, is_active, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = a.db.Exec(query, video.Link, video.Published_At, video.Title, video.Description, video.Thumbnail, video.Youtube_ID, video.Channel_Title, video.Channel_ID, userId, true, time.Now(), time.Now())

	if err != nil {
		fmt.Println("Error creating video", err)
		return err
	}

	query = `
		UPDATE video_requests
		SET processed_by = $1, processed_at = $2, status = 'ACCEPTED'
		WHERE id = $3
	`
	_, err = a.db.Exec(query, userId, time.Now(), requestID)
	if err != nil {
		fmt.Println("Error updating video request", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		fmt.Println("Error committing transaction", err)
		return err
	}

	return nil
}
