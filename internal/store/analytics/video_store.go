package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ClickhouseVideoStore struct {
	conn driver.Conn
}

func NewClickhouseVideoStore(conn driver.Conn) *ClickhouseVideoStore {
	return &ClickhouseVideoStore{conn: conn}
}

type VideoTimelineSnapshot struct {
	SnapshotTime time.Time `json:"snapshot_time"`
	Title        string    `json:"title"`
	ImageUrl     string    `json:"image_url"`
	Link         string    `json:"link"`
}

type AnalyticsVideoStore interface {
	GetVideoAnalyticsByID(videoID string) ([]VideoTimelineSnapshot, error)
}

func (c *ClickhouseVideoStore) GetVideoAnalyticsByID(videoID string) ([]VideoTimelineSnapshot, error) {

	query := `
		SELECT snapshot_time, title, link, image_url
		FROM default.video_snapshots
		WHERE video_id = ?
		ORDER BY snapshot_time DESC
	`

	rows, err := c.conn.Query(context.Background(), query, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get video analytics: %w", err)
	}
	defer rows.Close()

	var videos []VideoTimelineSnapshot

	for rows.Next() {
		var video VideoTimelineSnapshot

		err := rows.Scan(
			&video.SnapshotTime,
			&video.Title,
			&video.Link,
			&video.ImageUrl,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}
		videos = append(videos, video)
	}

	return videos, nil

}
