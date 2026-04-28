package model

import "time"

type Result struct {
	CreatorName     string
	ChannelID       string
	ChannelURL      string
	SubscriberCount uint64
	VideoTitle      string
	VideoID         string
	VideoURL        string
	ViewCount       uint64
	LikeCount       uint64
	CommentCount    uint64
	PublishedAt     time.Time
	Duration        string
	Format          string
	Language        string
	FilteredReason  string
	Description     string
}
