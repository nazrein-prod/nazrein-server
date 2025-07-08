
-- +goose Up
-- +goose StatementBegin

ALTER TABLE videos
ADD COLUMN visits INTEGER DEFAULT 0 CHECK (visits >= 0);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE videos
DROP COLUMN IF EXISTS visits;

-- +goose StatementEnd