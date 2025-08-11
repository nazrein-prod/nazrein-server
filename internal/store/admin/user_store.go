package admin

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/grvbrk/nazrein_server/internal/models"
)

type AdminPostgresUserStore struct {
	db *sql.DB
}

func NewPostgresAdminUserStore(db *sql.DB) *AdminPostgresUserStore {
	return &AdminPostgresUserStore{db: db}
}

type AdminUserStore interface {
	GetUserByID(UserID uuid.UUID) (*models.User, error)
}

func (a *AdminPostgresUserStore) GetUserByID(UserID uuid.UUID) (*models.User, error) {
	user := &models.User{}

	query := `
		SELECT id, role, name, image, videos_tracked
		FROM users
		WHERE id = $1
	`

	err := a.db.QueryRow(query, UserID).Scan(&user.ID, &user.Role, &user.Name, &user.ImageSrc, &user.Videos_Tracked)
	if err != nil {
		return nil, fmt.Errorf("failed to select user: %w", err)
	}

	return user, nil
}
