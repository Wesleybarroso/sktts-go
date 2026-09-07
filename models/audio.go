package models

import "time"

type AudioFile struct {
	ID          string    `json:"id"`
	KeyID       string    `json:"key_id"`
	Filename    string    `json:"filename"`
	Voice       string    `json:"voice"`
	Format      string    `json:"format"`
	ContentType string    `json:"content_type"`
	Characters  int       `json:"characters"`
	SizeBytes   int64     `json:"size_bytes"`
	Path        string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
}
