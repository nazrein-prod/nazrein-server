package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grvbrk/track-yt-video/internal/models"
)

type SortBy string

const (
	SortByPopular SortBy = "popular"
	SortByRecent  SortBy = "recent"
)

type GetVideosParams struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	SortBy SortBy `json:"sort_by"`
}

type VideosResponse struct {
	Videos  []VideoWithCounts `json:"videos"`
	Page    int               `json:"page"`
	Limit   int               `json:"limit"`
	Total   int               `json:"total"`
	HasMore bool              `json:"has_more"`
}

type VideoWithBookmarksResponse struct {
	Videos  []BookmarkedVideoWithCounts `json:"videos"`
	Page    int                         `json:"page"`
	Limit   int                         `json:"limit"`
	Total   int                         `json:"total"`
	HasMore bool                        `json:"has_more"`
}

type VideoWithCounts struct {
	models.Video
	BookmarkCount int `json:"bookmark_count"`
}

type BookmarkedVideoWithCounts struct {
	VideoWithCounts
	IsBookmarked bool `json:"is_bookmarked"`
}

type BookmarkedVideo struct {
	models.Video
	Bookmarked_At time.Time `json:"bookmarked_at"`
}

type PostgresVideoStore struct {
	db *sql.DB
}

func NewPostgresVideoStore(db *sql.DB) *PostgresVideoStore {
	if db == nil {
		panic("db cannot be nil for PostgresVideoStore")
	}
	return &PostgresVideoStore{db: db}
}

type VideoStore interface {
	GetVideos(params GetVideosParams) (*VideosResponse, error)
	GetVideosWithUserBookmarks(params GetVideosParams, userID uuid.UUID) (*VideoWithBookmarksResponse, error)
	GetVideosByUserID(userID uuid.UUID) ([]models.Video, error)
	GetVideoByID(videoID uuid.UUID) (*VideoWithCounts, error)
	GetBookmarkedVideosByUserID(userID uuid.UUID) ([]BookmarkedVideo, error)
}

func (pg *PostgresVideoStore) GetVideos(params GetVideosParams) (*VideosResponse, error) {

	offset := (params.Page - 1) * params.Limit

	var orderBy string
	switch params.SortBy {
	case SortByRecent:
		orderBy = "ORDER BY v.created_at DESC"
	case SortByPopular:
		orderBy = "ORDER BY v.published_at DESC"
	default:
		orderBy = "ORDER BY v.published_at DESC"
	}

	query := `
		SELECT COUNT(*) FROM videos v
		WHERE v.is_active = true
	`

	var total int
	err := pg.db.QueryRow(query).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total video count: %w", err)
	}

	query = fmt.Sprintf(`
		SELECT
			v.id,
			v.link,
			v.published_at,
			v.title,
			v.description,
			v.thumbnail,
			v.youtube_id,
			v.channel_title,
			v.channel_id,
			v.user_id,
			v.is_active,
			v.visits,
			v.created_at,
			v.updated_at,
			COALESCE(b.bookmark_count, 0) as bookmark_count
		FROM videos v
		LEFT JOIN (
			SELECT video_id, COUNT(*) as bookmark_count
			FROM bookmarks
			GROUP BY video_id
		) b ON v.id = b.video_id
		WHERE v.is_active = true
		%s
		LIMIT %d OFFSET %d
	`, orderBy, params.Limit, offset)

	rows, err := pg.db.Query(query)

	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}
	defer rows.Close()

	var videos []VideoWithCounts
	for rows.Next() {
		var v VideoWithCounts

		err := rows.Scan(
			&v.Id,
			&v.Link,
			&v.Published_At,
			&v.Title,
			&v.Description,
			&v.Thumbnail,
			&v.Youtube_ID,
			&v.Channel_Title,
			&v.Channel_ID,
			&v.User_ID,
			&v.Is_Active,
			&v.Visits,
			&v.Created_At,
			&v.Updated_At,
			&v.BookmarkCount,
		)
		if err != nil {
			return nil, err
		}
		videos = append(videos, v)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over video rows: %w", err)
	}

	hasMore := offset+len(videos) < total

	return &VideosResponse{
		Videos:  videos,
		Page:    params.Page,
		Limit:   params.Limit,
		Total:   total,
		HasMore: hasMore,
	}, nil
}

func (pg *PostgresVideoStore) GetVideosWithUserBookmarks(params GetVideosParams, userID uuid.UUID) (*VideoWithBookmarksResponse, error) {
	offset := (params.Page - 1) * params.Limit

	var orderBy string
	switch params.SortBy {
	case SortByRecent:
		orderBy = "ORDER BY v.created_at DESC"
	case SortByPopular:
		orderBy = "ORDER BY v.published_at DESC"
	default:
		orderBy = "ORDER BY v.published_at DESC"
	}

	// Get total count
	query := `
		SELECT COUNT(*) FROM videos v
		WHERE v.is_active = true
	`

	var total int
	err := pg.db.QueryRow(query).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total video count: %w", err)
	}

	// Get videos with bookmark information
	query = fmt.Sprintf(`
    SELECT
        v.id,
        v.link,
        v.published_at,
        v.title,
        v.description,
        v.thumbnail,
        v.youtube_id,
        v.channel_title,
        v.channel_id,
        v.user_id,
        v.is_active,
        v.visits,
        v.created_at,
        v.updated_at,
        COALESCE(bc.bookmark_count, 0) AS bookmark_count,
        CASE WHEN bu.id IS NOT NULL THEN true ELSE false END AS is_bookmarked
    FROM videos v
    LEFT JOIN bookmarks bu ON v.id = bu.video_id AND bu.user_id = $1
    LEFT JOIN (
        SELECT video_id, COUNT(*) AS bookmark_count
        FROM bookmarks
        GROUP BY video_id
    ) bc ON v.id = bc.video_id
    WHERE v.is_active = true
    %s
    LIMIT %d OFFSET %d
`, orderBy, params.Limit, offset)

	rows, err := pg.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos with bookmarks: %w", err)
	}
	defer rows.Close()

	var videos []BookmarkedVideoWithCounts
	for rows.Next() {
		var video BookmarkedVideoWithCounts

		err := rows.Scan(
			&video.Id,
			&video.Link,
			&video.Published_At,
			&video.Title,
			&video.Description,
			&video.Thumbnail,
			&video.Youtube_ID,
			&video.Channel_Title,
			&video.Channel_ID,
			&video.User_ID,
			&video.Is_Active,
			&video.Visits,
			&video.Created_At,
			&video.Updated_At,
			&video.BookmarkCount,
			&video.IsBookmarked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video with bookmark: %w", err)
		}
		videos = append(videos, video)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over video rows: %w", err)
	}

	hasMore := offset+len(videos) < total

	return &VideoWithBookmarksResponse{
		Videos:  videos,
		Page:    params.Page,
		Limit:   params.Limit,
		Total:   total,
		HasMore: hasMore,
	}, nil
}

func (pg *PostgresVideoStore) GetVideosByUserID(userId uuid.UUID) ([]models.Video, error) {

	query := `
	SELECT *
	FROM videos v
	WHERE v.user_id = $1
	`

	rows, err := pg.db.Query(query, userId)
	if err != nil {
		return nil, fmt.Errorf("failed to get videos: %w", err)
	}

	defer rows.Close()

	var videos []models.Video
	for rows.Next() {
		var video models.Video

		err := rows.Scan(
			&video.Id,
			&video.Link,
			&video.Published_At,
			&video.Title,
			&video.Description,
			&video.Thumbnail,
			&video.Youtube_ID,
			&video.Channel_Title,
			&video.Channel_ID,
			&video.User_ID,
			&video.Is_Active,
			&video.Visits,
			&video.Created_At,
			&video.Updated_At,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}
		videos = append(videos, video)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over video rows: %w", err)
	}

	return videos, nil
}

func (pg *PostgresVideoStore) GetVideoByID(videoID uuid.UUID) (*VideoWithCounts, error) {

	tx, err := pg.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}

	defer tx.Rollback()

	query := `
		UPDATE videos
		SET visits = visits + 1
		WHERE id = $1
	`

	_, err = tx.Exec(query, videoID)
	if err != nil {
		return nil, fmt.Errorf("failed to update video visits: %w", err)
	}

	query = `
	SELECT
		v.id,
		v.link,
		v.published_at,
		v.title,
		v.description,
		v.thumbnail,
		v.youtube_id,
		v.channel_title,
		v.channel_id,
		v.user_id,
		v.is_active,
		v.visits,
		v.created_at,
		v.updated_at,
		COALESCE(b.bookmark_count, 0) as bookmark_count
	FROM videos v
	LEFT JOIN (
		SELECT video_id, COUNT(*) as bookmark_count
		FROM bookmarks
		GROUP BY video_id
	) b ON v.id = b.video_id
	WHERE v.id = $1
	`

	row := tx.QueryRow(query, videoID)

	var video VideoWithCounts
	err = row.Scan(
		&video.Id,
		&video.Link,
		&video.Published_At,
		&video.Title,
		&video.Description,
		&video.Thumbnail,
		&video.Youtube_ID,
		&video.Channel_Title,
		&video.Channel_ID,
		&video.User_ID,
		&video.Is_Active,
		&video.Visits,
		&video.Created_At,
		&video.Updated_At,
		&video.BookmarkCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan video: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &video, nil
}

func (pg *PostgresVideoStore) GetBookmarkedVideosByUserID(userID uuid.UUID) ([]BookmarkedVideo, error) {

	query := `
		SELECT
			v.id,
			v.link,
			v.title,
			v.description,
			v.thumbnail,
			v.youtube_id,
			v.channel_title,
			v.channel_id,
			v.published_at,
			v.is_active,
			v.created_at,
			v.updated_at,
			b.created_at as bookmarked_at
		FROM videos v
		INNER JOIN bookmarks b ON v.id = b.video_id
		WHERE b.user_id = $1 AND v.is_active = true
		ORDER BY b.created_at DESC;
	`

	rows, err := pg.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bookmarked videos: %w", err)
	}
	defer rows.Close()

	var bookmarkedVideos []BookmarkedVideo

	for rows.Next() {
		var bookmarkedVideo BookmarkedVideo

		err := rows.Scan(
			&bookmarkedVideo.Id,
			&bookmarkedVideo.Link,
			&bookmarkedVideo.Title,
			&bookmarkedVideo.Description,
			&bookmarkedVideo.Thumbnail,
			&bookmarkedVideo.Youtube_ID,
			&bookmarkedVideo.Channel_Title,
			&bookmarkedVideo.Channel_ID,
			&bookmarkedVideo.Published_At,
			&bookmarkedVideo.Is_Active,
			&bookmarkedVideo.Created_At,
			&bookmarkedVideo.Updated_At,
			&bookmarkedVideo.Bookmarked_At,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan video: %w", err)
		}

		bookmarkedVideos = append(bookmarkedVideos, bookmarkedVideo)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over video rows: %w", err)
	}

	return bookmarkedVideos, nil
}

func ValidateSortBy(sortBy string) SortBy {
	switch SortBy(sortBy) {
	case SortByPopular:
		return SortByPopular
	case SortByRecent:
		return SortByRecent
	default:
		return SortByPopular // Default to popular
	}
}
