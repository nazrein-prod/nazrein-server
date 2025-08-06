-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE videos
ADD COLUMN search_vector tsvector,
ADD COLUMN normalized_video_title TEXT,
ADD COLUMN normalized_channel_title TEXT;

CREATE OR REPLACE FUNCTION update_video_search_vector() RETURNS TRIGGER AS $$
BEGIN

    NEW.normalized_video_title = lower(trim(regexp_replace(NEW.title, '\s+', ' ', 'g')));
    NEW.normalized_channel_title = lower(trim(regexp_replace(NEW.channel_title, '\s+', ' ', 'g')));

    -- A = higher weight (video title), B = lower weight (channel name )
    NEW.search_vector =
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.channel_title, '')), 'B');

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_video_search_vector_trigger
    BEFORE INSERT OR UPDATE OF title, channel_title, description ON videos
    FOR EACH ROW
    EXECUTE FUNCTION update_video_search_vector();

-- This forces the above trigger to go through all the rows
-- Basically updates the existing rows with the new columns
UPDATE videos SET title = title WHERE TRUE;

CREATE INDEX idx_videos_search_vector ON videos USING gin(search_vector);
CREATE INDEX idx_videos_normalized_video_title ON videos USING gin(normalized_video_title gin_trgm_ops);
CREATE INDEX idx_videos_normalized_channel ON videos USING gin(normalized_channel_title gin_trgm_ops);
CREATE INDEX idx_videos_title_trigram ON videos USING gin(title gin_trgm_ops);
CREATE INDEX idx_videos_channel_trigram ON videos USING gin(channel_title gin_trgm_ops);

CREATE INDEX idx_videos_active_search ON videos(is_active) WHERE is_active = true;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS update_video_search_vector_trigger ON videos;
DROP FUNCTION IF EXISTS update_video_search_vector();

DROP INDEX IF EXISTS idx_videos_search_vector;
DROP INDEX IF EXISTS idx_videos_normalized_video_title;
DROP INDEX IF EXISTS idx_videos_normalized_channel;
DROP INDEX IF EXISTS idx_videos_title_trigram;
DROP INDEX IF EXISTS idx_videos_channel_trigram;
DROP INDEX IF EXISTS idx_videos_active_search;

ALTER TABLE videos
DROP COLUMN IF EXISTS search_vector,
DROP COLUMN IF EXISTS normalized_video_title,
DROP COLUMN IF EXISTS normalized_channel_title;

DROP EXTENSION IF EXISTS pg_trgm;

-- +goose StatementEnd