package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"sktts-go/database"
	"sktts-go/middleware"
	"sktts-go/models"
)

type KeyService struct {
	DB *database.DB
}

func NewKeyService(db *database.DB) *KeyService {
	return &KeyService{DB: db}
}

func GenerateKey() (string, error) {
	buf := make([]byte, 32)

	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("gerar chave: %w", err)
	}

	return "sk_" + hex.EncodeToString(buf), nil
}

func (s *KeyService) Create(
	name string,
	keyType string,
	plan string,
	expiresAt *time.Time,
) (*models.APIKey, string, error) {

	rawKey, err := GenerateKey()
	if err != nil {
		return nil, "", err
	}

	now := time.Now().UTC()

	item := &models.APIKey{
		ID:        uuid.NewString(),
		KeyHash:   middleware.HashKey(rawKey),
		Name:      name,
		KeyType:   keyType,
		Plan:      plan,
		Status:    "active",
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	_, err = s.DB.SQL.Exec(`
		INSERT INTO api_keys
		(id, key_hash, name, key_type, plan, status, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.KeyHash,
		item.Name,
		item.KeyType,
		item.Plan,
		item.Status,
		item.ExpiresAt,
		item.CreatedAt,
	)

	if err != nil {
		return nil, "", fmt.Errorf("salvar chave: %w", err)
	}

	item.KeyHash = ""

	return item, rawKey, nil
}

func (s *KeyService) List() ([]models.APIKey, error) {
	rows, err := s.DB.SQL.Query(`
		SELECT id, name, key_type, plan, status,
		       expires_at, created_at, revoked_at, suspended_at
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listar chaves: %w", err)
	}
	defer rows.Close()

	var keys []models.APIKey

	for rows.Next() {
		var item models.APIKey

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.KeyType,
			&item.Plan,
			&item.Status,
			&item.ExpiresAt,
			&item.CreatedAt,
			&item.RevokedAt,
			&item.SuspendedAt,
		); err != nil {
			return nil, fmt.Errorf("ler chave: %w", err)
		}

		keys = append(keys, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar chaves: %w", err)
	}

	return keys, nil
}

func (s *KeyService) Get(id string) (*models.APIKey, error) {
	var item models.APIKey

	err := s.DB.SQL.QueryRow(`
		SELECT id, name, key_type, plan, status,
		       expires_at, created_at, revoked_at, suspended_at
		FROM api_keys
		WHERE id = ?
	`, id).Scan(
		&item.ID,
		&item.Name,
		&item.KeyType,
		&item.Plan,
		&item.Status,
		&item.ExpiresAt,
		&item.CreatedAt,
		&item.RevokedAt,
		&item.SuspendedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("buscar chave: %w", err)
	}

	return &item, nil
}

func (s *KeyService) Suspend(id string) error {
	now := time.Now().UTC()

	result, err := s.DB.SQL.Exec(`
		UPDATE api_keys
		SET status = 'suspended',
		    suspended_at = ?,
		    revoked_at = NULL
		WHERE id = ?
		  AND status != 'deleted'
	`, now, id)

	if err != nil {
		return fmt.Errorf("suspender chave: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar suspensão: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("chave não encontrada ou não pode ser suspensa")
	}

	return nil
}

func (s *KeyService) Activate(id string) error {
	result, err := s.DB.SQL.Exec(`
		UPDATE api_keys
		SET status = 'active',
		    suspended_at = NULL
		WHERE id = ?
		  AND status = 'suspended'
	`, id)

	if err != nil {
		return fmt.Errorf("ativar chave: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar ativação: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("chave não encontrada ou não está suspensa")
	}

	return nil
}

func (s *KeyService) Revoke(id string) error {
	now := time.Now().UTC()

	result, err := s.DB.SQL.Exec(`
		UPDATE api_keys
		SET status = 'revoked',
		    revoked_at = ?,
		    suspended_at = NULL
		WHERE id = ?
		  AND status NOT IN ('deleted', 'revoked')
	`, now, id)

	if err != nil {
		return fmt.Errorf("revogar chave: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar revogação: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("chave não encontrada ou já revogada")
	}

	return nil
}

func (s *KeyService) Delete(id string) error {
	result, err := s.DB.SQL.Exec(`
		UPDATE api_keys
		SET status = 'deleted'
		WHERE id = ?
		  AND status != 'deleted'
	`, id)

	if err != nil {
		return fmt.Errorf("deletar chave: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar deleção: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("chave não encontrada ou já deletada")
	}

	return nil
}
