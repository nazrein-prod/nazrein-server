package store

import (
	"database/sql"
	"fmt"

	"github.com/grvbrk/nazrein_server/internal/models"
)

type PostgresUserStore struct {
	db *sql.DB
}

func NewPostgresUserStore(db *sql.DB) *PostgresUserStore {
	return &PostgresUserStore{db: db}
}

type UserStore interface {
	CreateUser(*models.User) error
	GetUserByUsername(username string) (*models.User, error)
	GetUserByGoogleID(id string) (*models.User, error)
}

func (pg *PostgresUserStore) CreateUser(user *models.User) error {

	query := `
	INSERT INTO users (google_id, name, email, image, role)
	VALUES ($1, $2, $3, $4, $5)
	RETURNING id;
	`
	err := pg.db.QueryRow(query, user.GoogleID, user.Name, user.Email, user.ImageSrc, user.Role).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("error running create user query: %w", err)
	}

	return nil
}

func (pg *PostgresUserStore) GetUserByUsername(username string) (*models.User, error) {
	user := &models.User{}

	query := `
	SELECT id, google_id, name, email, image, role
	FROM users
	WHERE name = $1;
	`

	err := pg.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.GoogleID,
		&user.Name,
		&user.Email,
		&user.ImageSrc,
		&user.Role,
	)

	if err != nil {
		return nil, fmt.Errorf("error running get user by username query: %w", err)
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no user found with username: %s", username)
	}

	return user, nil
}

func (pg *PostgresUserStore) GetUserByGoogleID(id string) (*models.User, error) {
	user := &models.User{}

	query := `
	SELECT id, google_id, name, email, image, role
	FROM users
	WHERE google_id = $1
	`

	err := pg.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.GoogleID,
		&user.Name,
		&user.Email,
		&user.ImageSrc,
		&user.Role,
	)

	if err != nil {
		return nil, fmt.Errorf("error running get user by google id query: %w", err)
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no user found with google id: %s", id)
	}

	return user, nil
}
