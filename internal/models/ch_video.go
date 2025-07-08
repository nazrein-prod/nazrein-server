package models

import "time"

type ClickhouseVideo struct {
	VideoID      string    `ch:"video_id"`
	SnapshotTime time.Time `ch:"snapshot_time"`
	Title        string    `ch:"title"`
	ImageSrc     string    `ch:"image_src"`
	Link         string    `ch:"link"`
	TitleHash    uint64    `ch:"title_hash"`
	ImageEtag    string    `ch:"image_etag"`
	CreatedAt    time.Time `ch:"created_at"`
}
