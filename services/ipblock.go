package services

import (
	"fmt"
	"time"

	"sktts-go/database"
)

type IPBlockService struct {
	DB *database.DB
}

func NewIPBlockService(db *database.DB) *IPBlockService {
	return &IPBlockService{DB: db}
}

func (s *IPBlockService) IsBlocked(ip string) (bool, error) {
	var blockedUntil time.Time

	err := s.DB.SQL.QueryRow(`
		SELECT blocked_until
		FROM ip_blocks
		WHERE ip = ?
	`, ip).Scan(&blockedUntil)

	if err != nil {
		return false, nil
	}

	if time.Now().UTC().Before(blockedUntil) {
		return true, nil
	}

	_, err = s.DB.SQL.Exec(`
		DELETE FROM ip_blocks
		WHERE ip = ?
	`, ip)

	if err != nil {
		return false, fmt.Errorf("remover bloqueio expirado: %w", err)
	}

	return false, nil
}

func (s *IPBlockService) Block(ip, reason string) error {
	blockedUntil := time.Now().UTC().Add(24 * time.Hour)

	_, err := s.DB.SQL.Exec(`
		INSERT INTO ip_blocks (ip, reason, blocked_until)
		VALUES (?, ?, ?)
		ON CONFLICT(ip)
		DO UPDATE SET
			reason = excluded.reason,
			blocked_until = excluded.blocked_until
	`,
		ip,
		reason,
		blockedUntil,
	)

	if err != nil {
		return fmt.Errorf("bloquear IP: %w", err)
	}

	return nil
}

func (s *IPBlockService) Unblock(ip string) error {
	_, err := s.DB.SQL.Exec(`
		DELETE FROM ip_blocks
		WHERE ip = ?
	`, ip)

	if err != nil {
		return fmt.Errorf("desbloquear IP: %w", err)
	}

	return nil
}
