package admin

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AdminPostgresVideoRequestStore struct {
	db *sql.DB
}

func NewPostgresAdminVideoRequestStore(db *sql.DB) *AdminPostgresVideoRequestStore {
	return &AdminPostgresVideoRequestStore{db: db}
}

type AdminVideoRequestStore interface {
	DeleteVideoRequest(requestID uuid.UUID) error
	PatchVideoRequest(requestID uuid.UUID, status *string, processedBy *string, rejectionReason *string) error
}

func (a *AdminPostgresVideoRequestStore) DeleteVideoRequest(requestID uuid.UUID) error {
	query := `
	DELETE FROM video_requests
	WHERE id = $1
	`

	_, err := a.db.Exec(query, requestID)
	if err != nil {
		return fmt.Errorf("ADMIN: failed to delete video request: %w", err)
	}

	return nil
}

func (a *AdminPostgresVideoRequestStore) PatchVideoRequest(
	requestID uuid.UUID,
	status *string,
	processedBy *string,
	rejectionReason *string,
) error {

	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argPos))
		args = append(args, *status)
		argPos++
	}
	if processedBy != nil {
		setClauses = append(setClauses, fmt.Sprintf("processed_by = $%d", argPos))
		args = append(args, *processedBy)
		argPos++
	}
	if rejectionReason != nil {
		setClauses = append(setClauses, fmt.Sprintf("rejection_reason = $%d", argPos))
		args = append(args, *rejectionReason)
		argPos++
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no fields provided to update")
	}

	setClauses = append(setClauses, fmt.Sprintf("processed_at = $%d", argPos))
	args = append(args, time.Now())
	argPos++

	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")

	query := fmt.Sprintf(`
		UPDATE video_requests
		SET %s
		WHERE id = $%d
	`, strings.Join(setClauses, ", "), argPos)

	args = append(args, requestID)

	_, err := a.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("ADMIN: failed to update video request: %w", err)
	}

	return nil
}
