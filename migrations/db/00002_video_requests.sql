-- +goose Up
-- +goose StatementBegin

CREATE TYPE video_request_status AS ENUM ('PENDING', 'ACCEPTED', 'REJECTED');

CREATE TABLE IF NOT EXISTS video_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  status video_request_status DEFAULT 'PENDING',
  link VARCHAR(255) UNIQUE NOT NULL,
  youtube_id VARCHAR(20) UNIQUE NOT NULL,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  processed_by UUID REFERENCES users(id),
  processed_at TIMESTAMP WITH TIME ZONE,
  rejection_reason TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

  UNIQUE(youtube_id, user_id)
);

CREATE INDEX idx_video_requests_user_status ON video_requests(user_id, status);
CREATE INDEX idx_video_requests_youtube_id ON video_requests(youtube_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_video_requests_user_status;
DROP INDEX IF EXISTS idx_video_requests_youtube_id;

DROP TABLE IF EXISTS video_requests;

DROP TYPE IF EXISTS video_request_status;

-- +goose StatementEnd