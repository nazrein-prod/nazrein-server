package store

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Dashboard struct {
	Bookmarked int `json:"bookmarked"`
	Tracked    int `json:"tracked"`
	Pending    int `json:"pending"`
}

type PostgresDashboardStore struct {
	db *sql.DB
}

func NewPostgresDashboardStore(db *sql.DB) *PostgresDashboardStore {
	return &PostgresDashboardStore{db: db}
}

type DashboardStore interface {
	GetDashboardMetricsByUserID(userID uuid.UUID) (*Dashboard, error)
}

func (pg *PostgresDashboardStore) GetDashboardMetricsByUserID(userID uuid.UUID) (*Dashboard, error) {

	var dashboard Dashboard

	query := `
		SELECT
			(SELECT COUNT(*) FROM bookmarks WHERE user_id = $1) as bookmarked_videos,
			(SELECT videos_tracked FROM users WHERE id = $1) as total_tracked_videos,
			(SELECT COUNT(*) FROM video_requests WHERE user_id = $1 AND status = 'PENDING') as pending_requests;
	`

	err := pg.db.QueryRow(query, userID).Scan(&dashboard.Bookmarked, &dashboard.Tracked, &dashboard.Pending)
	if err != nil {
		return nil, fmt.Errorf("error getting dashboard metrics: %w", err)
	}

	return &dashboard, nil
}
