CREATE TABLE IF NOT EXISTS default.video_snapshots (
  video_id String,
  youtube_id String,
  snapshot_time DateTime,
  title String,
  image_src String,
  link String,
  title_hash UInt64,
  image_etag String,
  image_file_id String,
  image_filename String,
  image_url String,
  image_thumbnail_url String,
  image_height int,
  image_width int,
  image_size UInt64,
  image_filepath String,

  created_at DateTime DEFAULT now()
)
ENGINE = MergeTree()
ORDER BY (video_id, snapshot_time)
PARTITION BY toYYYYMM(snapshot_time);

