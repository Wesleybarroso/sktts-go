package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"sktts-go/database"
	"sktts-go/models"
)

type PlansService struct {
	DB *database.DB
}

func NewPlansService(db *database.DB) *PlansService {
	return &PlansService{DB: db}
}

func (s *PlansService) Create(
	name string,
	description string,
	price float64,
	requestsPerMinute int,
	requestsPerDay int,
	charactersPerRequest int,
	charactersPerDay int,
	maxConcurrentTTS int,
) (*models.Plan, error) {

	now := time.Now().UTC()

	item := &models.Plan{
		ID:                   uuid.NewString(),
		Name:                 name,
		Description:          description,
		Price:                price,
		RequestsPerMinute:    requestsPerMinute,
		RequestsPerDay:       requestsPerDay,
		CharactersPerRequest: charactersPerRequest,
		CharactersPerDay:     charactersPerDay,
		MaxConcurrentTTS:     maxConcurrentTTS,
		Status:               "active",
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	_, err := s.DB.SQL.Exec(`
		INSERT INTO plans
		(id, name, description, price,
		 requests_per_minute, requests_per_day,
		 characters_per_request, characters_per_day,
		 max_concurrent_tts, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		item.ID,
		item.Name,
		item.Description,
		item.Price,
		item.RequestsPerMinute,
		item.RequestsPerDay,
		item.CharactersPerRequest,
		item.CharactersPerDay,
		item.MaxConcurrentTTS,
		item.Status,
		item.CreatedAt,
		item.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("salvar plano: %w", err)
	}

	return item, nil
}

func (s *PlansService) List() ([]models.Plan, error) {
	rows, err := s.DB.SQL.Query(`
		SELECT id, name, description, price,
		       requests_per_minute, requests_per_day,
		       characters_per_request, characters_per_day,
		       max_concurrent_tts, status,
		       created_at, updated_at
		FROM plans
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listar planos: %w", err)
	}
	defer rows.Close()

	var plans []models.Plan

	for rows.Next() {
		var item models.Plan

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.RequestsPerMinute,
			&item.RequestsPerDay,
			&item.CharactersPerRequest,
			&item.CharactersPerDay,
			&item.MaxConcurrentTTS,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("ler plano: %w", err)
		}

		plans = append(plans, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterar planos: %w", err)
	}

	return plans, nil
}

func (s *PlansService) Get(id string) (*models.Plan, error) {
	var item models.Plan

	err := s.DB.SQL.QueryRow(`
		SELECT id, name, description, price,
		       requests_per_minute, requests_per_day,
		       characters_per_request, characters_per_day,
		       max_concurrent_tts, status,
		       created_at, updated_at
		FROM plans
		WHERE id = ?
	`, id).Scan(
		&item.ID,
		&item.Name,
		&item.Description,
		&item.Price,
		&item.RequestsPerMinute,
		&item.RequestsPerDay,
		&item.CharactersPerRequest,
		&item.CharactersPerDay,
		&item.MaxConcurrentTTS,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("buscar plano: %w", err)
	}

	return &item, nil
}

func (s *PlansService) Update(
	id string,
	name string,
	description string,
	price float64,
	requestsPerMinute int,
	requestsPerDay int,
	charactersPerRequest int,
	charactersPerDay int,
	maxConcurrentTTS int,
	status string,
) (*models.Plan, error) {

	now := time.Now().UTC()

	result, err := s.DB.SQL.Exec(`
		UPDATE plans
		SET name = ?,
		    description = ?,
		    price = ?,
		    requests_per_minute = ?,
		    requests_per_day = ?,
		    characters_per_request = ?,
		    characters_per_day = ?,
		    max_concurrent_tts = ?,
		    status = ?,
		    updated_at = ?
		WHERE id = ?
	`,
		name,
		description,
		price,
		requestsPerMinute,
		requestsPerDay,
		charactersPerRequest,
		charactersPerDay,
		maxConcurrentTTS,
		status,
		now,
		id,
	)

	if err != nil {
		return nil, fmt.Errorf("atualizar plano: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("verificar atualização: %w", err)
	}

	if affected == 0 {
		return nil, fmt.Errorf("plano não encontrado")
	}

	return s.Get(id)
}

func (s *PlansService) Delete(id string) error {
	result, err := s.DB.SQL.Exec(`
		UPDATE plans
		SET status = 'deleted',
		    updated_at = ?
		WHERE id = ?
		  AND status != 'deleted'
	`, time.Now().UTC(), id)

	if err != nil {
		return fmt.Errorf("deletar plano: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("verificar deleção: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("plano não encontrado ou já deletado")
	}

	return nil
}
