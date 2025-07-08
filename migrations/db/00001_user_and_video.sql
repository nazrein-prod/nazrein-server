-- +goose Up
-- +goose StatementBegin

CREATE TYPE user_role AS ENUM ('ADMIN', 'USER');

CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  google_id VARCHAR(255) UNIQUE NOT NULL,
  name VARCHAR(50) NOT NULL,
  email VARCHAR(255) UNIQUE NOT NULL,
  image VARCHAR(255),
  role user_role DEFAULT 'USER',
  videos_tracked INTEGER DEFAULT 0 CHECK (videos_tracked >= 0),
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS videos (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  link VARCHAR(255) UNIQUE NOT NULL,
  published_at TIMESTAMP WITH TIME ZONE,
  title VARCHAR(255) NOT NULL,
  description TEXT,
  thumbnail VARCHAR(255) NOT NULL,
  youtube_id VARCHAR(20) UNIQUE NOT NULL,
  channel_title VARCHAR(255) NOT NULL,
  channel_id VARCHAR(255),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  is_active BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

  UNIQUE(youtube_id, user_id)
);

CREATE INDEX idx_videos_user_active ON videos(user_id, is_active);
CREATE INDEX idx_videos_youtube_id ON videos(youtube_id);

CREATE OR REPLACE FUNCTION update_videos_tracked_count_of_user() RETURNS TRIGGER as $$
BEGIN
  IF (TG_OP = 'INSERT') AND NEW.is_active = TRUE THEN
    UPDATE users SET videos_tracked = videos_tracked + 1 WHERE id = NEW.user_id;
    RETURN NEW;
  ELSIF (TG_OP = 'UPDATE') THEN
    IF OLD.is_active = FALSE AND NEW.is_active = TRUE THEN
      UPDATE users SET videos_tracked = videos_tracked + 1 WHERE id = NEW.user_id;
    ELSIF OLD.is_active = TRUE AND NEW.is_active = FALSE THEN
      UPDATE users SET videos_tracked = videos_tracked - 1 WHERE id = NEW.user_id;
    END IF;
    RETURN NEW;
  ELSIF (TG_OP = 'DELETE') AND OLD.is_active = TRUE THEN
    UPDATE users SET videos_tracked = videos_tracked - 1 WHERE id = OLD.user_id;
    RETURN OLD;
  END IF;
  RETURN COALESCE(NEW, OLD);

END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_track_count
  AFTER INSERT OR UPDATE OR DELETE ON videos
  FOR EACH ROW
  EXECUTE FUNCTION update_videos_tracked_count_of_user();


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS update_track_count ON videos;
DROP FUNCTION IF EXISTS update_videos_tracked_count_of_user();

DROP INDEX IF EXISTS idx_videos_user_active;
DROP INDEX IF EXISTS idx_videos_youtube_id;

DROP TABLE IF EXISTS videos;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS user_role;

-- +goose StatementEnd