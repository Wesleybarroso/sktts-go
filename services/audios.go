package services

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"sktts-go/models"
)

const AudioStoragePath = "/opt/xtts/audio"

type AudioService struct {
	db *sql.DB
}

func NewAudioService(db *sql.DB) *AudioService {
	return &AudioService{db: db}
}

func (s *AudioService) Create(
	keyID string,
	filename string,
	voice string,
	format string,
	contentType string,
	characters int,
	data []byte,
) (*models.AudioFile, error) {

	if keyID == "" {
		return nil, errors.New("key_id obrigatório")
	}

	if len(data) == 0 {
		return nil, errors.New("áudio vazio")
	}

	if err := os.MkdirAll(AudioStoragePath, 0750); err != nil {
		return nil, fmt.Errorf("criar storage: %w", err)
	}

	id := uuid.New().String()

	// Nunca usamos nome fornecido pelo cliente como caminho.
	storageFilename := id + filepath.Ext(filename)

	fullPath := filepath.Join(AudioStoragePath, storageFilename)

	if err := os.WriteFile(fullPath, data, 0640); err != nil {
		return nil, fmt.Errorf("salvar áudio: %w", err)
	}

	now := time.Now().UTC()

	item := &models.AudioFile{
		ID:          id,
		KeyID:       keyID,
		Filename:    filename,
		Voice:       voice,
		Format:      format,
		ContentType: contentType,
		Characters:  characters,
		SizeBytes:   int64(len(data)),
		Path:        fullPath,
		CreatedAt:   now,
	}

	_, err := s.db.Exec(`
		INSERT INTO audio_files (
			id,
			key_id,
			filename,
			voice,
			format,
			content_type,
			characters,
			size_bytes,
			path,
			created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.KeyID,
		item.Filename,
		item.Voice,
		item.Format,
		item.ContentType,
		item.Characters,
		item.SizeBytes,
		item.Path,
		item.CreatedAt,
	)

	if err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("registrar áudio: %w", err)
	}

	return item, nil
}

func (s *AudioService) List(keyID string) ([]models.AudioFile, error) {

	rows, err := s.db.Query(`
		SELECT
			id,
			key_id,
			filename,
			voice,
			format,
			content_type,
			characters,
			size_bytes,
			path,
			created_at
		FROM audio_files
		WHERE key_id = ?
		ORDER BY created_at DESC
	`, keyID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.AudioFile, 0)

	for rows.Next() {
		var item models.AudioFile

		if err := rows.Scan(
			&item.ID,
			&item.KeyID,
			&item.Filename,
			&item.Voice,
			&item.Format,
			&item.ContentType,
			&item.Characters,
			&item.SizeBytes,
			&item.Path,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *AudioService) Get(id string, keyID string) (*models.AudioFile, error) {

	var item models.AudioFile

	err := s.db.QueryRow(`
		SELECT
			id,
			key_id,
			filename,
			voice,
			format,
			content_type,
			characters,
			size_bytes,
			path,
			created_at
		FROM audio_files
		WHERE id = ? AND key_id = ?
	`, id, keyID).Scan(
		&item.ID,
		&item.KeyID,
		&item.Filename,
		&item.Voice,
		&item.Format,
		&item.ContentType,
		&item.Characters,
		&item.SizeBytes,
		&item.Path,
		&item.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, os.ErrNotExist
		}

		return nil, err
	}

	return &item, nil
}

func (s *AudioService) Delete(id string, keyID string) error {

	item, err := s.Get(id, keyID)
	if err != nil {
		return err
	}

	if err := os.Remove(item.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remover arquivo: %w", err)
	}

	result, err := s.db.Exec(`
		DELETE FROM audio_files
		WHERE id = ? AND key_id = ?
	`, id, keyID)

	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected == 0 {
		return os.ErrNotExist
	}

	return nil
}
