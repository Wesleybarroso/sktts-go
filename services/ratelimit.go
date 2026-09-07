package services

import (
	"sync"
	"time"
)

type rateWindow struct {
	Start    time.Time
	Requests int
}

type RateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateWindow
	global  rateWindow
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		windows: make(map[string]rateWindow),
	}
}

// Allow aplica um limite por chave dentro de uma janela de 1 minuto.
func (r *RateLimiter) Allow(key string, limit int) bool {
	if limit <= 0 {
		return true
	}

	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	window, exists := r.windows[key]

	if !exists || now.Sub(window.Start) >= time.Minute {
		r.windows[key] = rateWindow{
			Start:    now,
			Requests: 1,
		}
		return true
	}

	if window.Requests >= limit {
		return false
	}

	window.Requests++
	r.windows[key] = window

	return true
}

// AllowGlobal aplica um limite global dentro de uma janela de 1 hora.
func (r *RateLimiter) AllowGlobal(limit int) bool {
	if limit <= 0 {
		return true
	}

	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.global.Start.IsZero() ||
		now.Sub(r.global.Start) >= time.Hour {

		r.global = rateWindow{
			Start:    now,
			Requests: 1,
		}
		return true
	}

	if r.global.Requests >= limit {
		return false
	}

	r.global.Requests++

	return true
}
