-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS bookmarks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  video_id UUID NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

  -- Ensure a user can only bookmark a video once
  UNIQUE(user_id, video_id)
);

-- Indexes for efficient querying
CREATE INDEX idx_bookmarks_user_id ON bookmarks(user_id);
CREATE INDEX idx_bookmarks_video_id ON bookmarks(video_id);
CREATE INDEX idx_bookmarks_created_at ON bookmarks(created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_bookmarks_user_id;
DROP INDEX IF EXISTS idx_bookmarks_video_id;
DROP INDEX IF EXISTS idx_bookmarks_created_at;

DROP TABLE IF EXISTS bookmarks;

-- +goose StatementEnd