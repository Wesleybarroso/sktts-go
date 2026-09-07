package services

import (
	"fmt"
	"net/http"
	"time"
)

type QuotaService struct {
	Usage       *UsageService
	IPBlock     *IPBlockService
	RateLimiter *RateLimiter
	Plans       *PlansService
}

func NewQuotaService(usage *UsageService, ipBlock *IPBlockService, plans *PlansService) *QuotaService {
	return &QuotaService{
		Usage:       usage,
		IPBlock:     ipBlock,
		RateLimiter: NewRateLimiter(),
		Plans:       plans,
	}
}

func (q *QuotaService) Check(
	keyID string,
	plan string,
	ip string,
	characters int,
) (int, string, int) {

	if characters <= 0 {
		return http.StatusBadRequest, "texto inválido", 0
	}

	planData, err := q.Plans.Get(plan)
	if err != nil {
		return http.StatusInternalServerError, "erro ao consultar plano", 0
	}

	if planData.Status != "active" {
		return http.StatusForbidden, "plano inativo", planData.MaxConcurrentTTS
	}

	limits := PlanLimits{
		RequestsPerDay:    planData.RequestsPerDay,
		CharactersPerDay:  planData.CharactersPerDay,
		RequestsPerMinute: planData.RequestsPerMinute,
	}

	if planData.CharactersPerRequest > 0 &&
		characters > planData.CharactersPerRequest {
		return http.StatusBadRequest, fmt.Sprintf(
			"texto excede o limite máximo de %d caracteres por requisição",
			planData.CharactersPerRequest,
		), planData.MaxConcurrentTTS
	}

	// Admin não possui limite de consumo.
	if plan == "admin" {
		return 0, "", planData.MaxConcurrentTTS
	}

	blocked, err := q.IPBlock.IsBlocked(ip)
	if err != nil {
		return http.StatusInternalServerError, "erro ao verificar bloqueio", planData.MaxConcurrentTTS
	}

	if blocked {
		return http.StatusTooManyRequests, "IP bloqueado temporariamente", planData.MaxConcurrentTTS
	}

	usage, err := q.Usage.Get(keyID, ip)
	if err != nil {
		return http.StatusInternalServerError, "erro ao consultar uso", planData.MaxConcurrentTTS
	}

	if limits.RequestsPerDay > 0 &&
		usage.Requests >= limits.RequestsPerDay {

		_ = q.IPBlock.Block(
			ip,
			fmt.Sprintf("limite diário de requisições excedido: plano %s", plan),
		)

		return http.StatusTooManyRequests, "limite diário de requisições excedido", planData.MaxConcurrentTTS
	}

	if limits.CharactersPerDay > 0 &&
		usage.Characters+characters > limits.CharactersPerDay {

		_ = q.IPBlock.Block(
			ip,
			fmt.Sprintf("limite diário de caracteres excedido: plano %s", plan),
		)

		return http.StatusTooManyRequests, "limite diário de caracteres excedido", planData.MaxConcurrentTTS
	}

	// Limite de requisições por minuto por chave + IP.
	rateKey := keyID + ":" + ip

	if !q.RateLimiter.Allow(rateKey, limits.RequestsPerMinute) {
		return http.StatusTooManyRequests, "limite de requisições por minuto excedido", planData.MaxConcurrentTTS
	}

	// Limite global do acesso público LinkedIn: 100 requisições por hora.
	if plan == "linkedin" && !q.RateLimiter.AllowGlobal(100) {
		return http.StatusTooManyRequests, "limite global de requisições LinkedIn excedido", planData.MaxConcurrentTTS
	}

	return 0, "", planData.MaxConcurrentTTS
}

func IsExpired(expiresAt *time.Time) bool {
	if expiresAt == nil {
		return false
	}

	return time.Now().UTC().After(*expiresAt)
}
