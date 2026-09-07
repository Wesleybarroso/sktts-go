package models

import "time"

type APIKey struct {
	ID          string     `json:"id"`
	KeyHash     string     `json:"-"`
	Name        string     `json:"name"`
	KeyType     string     `json:"key_type"`
	Plan        string     `json:"plan"`
	Status      string     `json:"status"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	SuspendedAt *time.Time `json:"suspended_at,omitempty"`
}

type UsageDaily struct {
	KeyID      string    `json:"key_id"`
	IP         string    `json:"ip"`
	UsageDate  string    `json:"usage_date"`
	Requests   int       `json:"requests"`
	Characters int       `json:"characters"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type IPBlock struct {
	IP           string    `json:"ip"`
	Reason       string    `json:"reason"`
	BlockedUntil time.Time `json:"blocked_until"`
	CreatedAt    time.Time `json:"created_at"`
}

type Plan struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	Price                float64   `json:"price"`
	RequestsPerMinute    int       `json:"requests_per_minute"`
	RequestsPerDay       int       `json:"requests_per_day"`
	CharactersPerRequest int       `json:"characters_per_request"`
	CharactersPerDay     int       `json:"characters_per_day"`
	MaxConcurrentTTS     int       `json:"max_concurrent_tts"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}
