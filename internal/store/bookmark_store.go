package store

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type PostgresBookmarkStore struct {
	db *sql.DB
}

func NewPostgresBookmarkStore(db *sql.DB) *PostgresBookmarkStore {
	return &PostgresBookmarkStore{db: db}
}

type BookmarkStore interface {
	CreateBookmark(videoID uuid.UUID, userID uuid.UUID) error
	DeleteBookmark(videoID uuid.UUID, userID uuid.UUID) error
}

func (p *PostgresBookmarkStore) CreateBookmark(videoID uuid.UUID, userID uuid.UUID) error {
	query := `
		INSERT INTO bookmarks (video_id, user_id)
		VALUES ($1, $2)
	`
	_, err := p.db.Exec(query, videoID, userID)
	if err != nil {
		return fmt.Errorf("failed to insert bookmark: %w", err)
	}
	return nil
}

func (p *PostgresBookmarkStore) DeleteBookmark(videoID uuid.UUID, userID uuid.UUID) error {
	query := `
		DELETE FROM bookmarks
		WHERE video_id = $1 AND user_id = $2
	`
	_, err := p.db.Exec(query, videoID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete bookmark: %w", err)
	}
	return nil
}
