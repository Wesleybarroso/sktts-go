package services

import (
	"fmt"
	"time"

	"sktts-go/database"
)

type UsageService struct {
	DB *database.DB
}

func NewUsageService(db *database.DB) *UsageService {
	return &UsageService{DB: db}
}

type Usage struct {
	Requests   int
	Characters int
}

func (s *UsageService) Get(keyID, ip string) (Usage, error) {
	date := time.Now().UTC().Format("2006-01-02")

	var usage Usage

	err := s.DB.SQL.QueryRow(`
		SELECT requests, characters
		FROM usage_daily
		WHERE key_id = ? AND ip = ? AND usage_date = ?
	`,
		keyID,
		ip,
		date,
	).Scan(
		&usage.Requests,
		&usage.Characters,
	)

	if err != nil {
		return Usage{}, nil
	}

	return usage, nil
}

func (s *UsageService) Add(
	keyID string,
	ip string,
	characters int,
) error {

	date := time.Now().UTC().Format("2006-01-02")

	_, err := s.DB.SQL.Exec(`
		INSERT INTO usage_daily
			(key_id, ip, usage_date, requests, characters)
		VALUES (?, ?, ?, 1, ?)
		ON CONFLICT(key_id, ip, usage_date)
		DO UPDATE SET
			requests = requests + 1,
			characters = characters + excluded.characters,
			updated_at = CURRENT_TIMESTAMP
	`,
		keyID,
		ip,
		date,
		characters,
	)

	if err != nil {
		return fmt.Errorf("registrar uso: %w", err)
	}

	return nil
}
