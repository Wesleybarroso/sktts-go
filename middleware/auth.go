package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type AuthContext struct {
	KeyID      string
	KeyType    string
	Plan       string
	IsAdmin    bool
	IsLinkedIn bool
}

type contextKey string

const authContextKey contextKey = "sktss-auth"

type Auth struct {
	APIKey          string
	AdminKey        string
	LinkedInFreeKey string
	DB              *sql.DB
}

func NewAuth(
	apiKey string,
	adminKey string,
	linkedinFreeKey string,
	db *sql.DB,
) *Auth {
	return &Auth{
		APIKey:          apiKey,
		AdminKey:        adminKey,
		LinkedInFreeKey: linkedinFreeKey,
		DB:              db,
	}
}

func Get(r *http.Request) *AuthContext {
	value := r.Context().Value(authContextKey)

	if value == nil {
		return nil
	}

	auth, ok := value.(*AuthContext)
	if !ok {
		return nil
	}

	return auth
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func secureEqual(a, b string) bool {
	if a == "" || b == "" {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	return subtle.ConstantTimeCompare(
		[]byte(a),
		[]byte(b),
	) == 1
}

func HashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		providedKey := strings.TrimSpace(
			r.Header.Get("X-API-Key"),
		)

		if providedKey == "" {
			WriteJSON(
				w,
				http.StatusUnauthorized,
				map[string]any{
					"success": false,
					"error":   "api_key_required",
				},
			)
			return
		}

		// =====================================================
		// ADMIN
		// =====================================================

		if secureEqual(providedKey, a.AdminKey) {
			auth := &AuthContext{
				KeyType: "admin",
				Plan:    "admin",
				IsAdmin: true,
			}

			ctx := context.WithValue(
				r.Context(),
				authContextKey,
				auth,
			)

			next.ServeHTTP(
				w,
				r.WithContext(ctx),
			)

			return
		}

		// =====================================================
		// CHAVE LEGADA
		// =====================================================

		if secureEqual(providedKey, a.APIKey) {
			auth := &AuthContext{
				KeyID:   "legacy",
				KeyType: "client",
				Plan:    "free_24h",
			}

			ctx := context.WithValue(
				r.Context(),
				authContextKey,
				auth,
			)

			next.ServeHTTP(
				w,
				r.WithContext(ctx),
			)

			return
		}

		// =====================================================
		// LINKEDIN FREE
		// =====================================================

		if secureEqual(providedKey, a.LinkedInFreeKey) {
			auth := &AuthContext{
				KeyID:      "linkedin-free",
				KeyType:    "linkedin",
				Plan:       "linkedin",
				IsLinkedIn: true,
			}

			ctx := context.WithValue(
				r.Context(),
				authContextKey,
				auth,
			)

			next.ServeHTTP(
				w,
				r.WithContext(ctx),
			)

			return
		}

		// =====================================================
		// CLIENT KEYS
		// =====================================================

		if a.DB == nil {
			WriteJSON(
				w,
				http.StatusInternalServerError,
				map[string]any{
					"success": false,
					"error":   "authentication_unavailable",
				},
			)
			return
		}

		hash := HashKey(providedKey)

		var (
			id        string
			keyType   string
			plan      string
			status    string
			expiresAt sql.NullString
		)

		err := a.DB.QueryRow(`
			SELECT
				id,
				key_type,
				plan,
				status,
				expires_at
			FROM api_keys
			WHERE key_hash = ?
		`, hash).Scan(
			&id,
			&keyType,
			&plan,
			&status,
			&expiresAt,
		)

		if err == sql.ErrNoRows {
			WriteJSON(
				w,
				http.StatusUnauthorized,
				map[string]any{
					"success": false,
					"error":   "invalid_api_key",
				},
			)
			return
		}

		if err != nil {
			WriteJSON(
				w,
				http.StatusInternalServerError,
				map[string]any{
					"success": false,
					"error":   "authentication_error",
				},
			)
			return
		}

		if status != "active" {
			WriteJSON(
				w,
				http.StatusUnauthorized,
				map[string]any{
					"success": false,
					"error":   "api_key_inactive",
					"status":  status,
				},
			)
			return
		}

		// =====================================================
		// EXPIRAÇÃO
		// =====================================================

		if expiresAt.Valid && expiresAt.String != "" {
			expiration, err := parseTime(expiresAt.String)

			if err == nil && time.Now().UTC().After(expiration) {
				_, _ = a.DB.Exec(`
					UPDATE api_keys
					SET status = 'expired'
					WHERE id = ?
				`, id)

				WriteJSON(
					w,
					http.StatusUnauthorized,
					map[string]any{
						"success": false,
						"error":   "api_key_expired",
					},
				)
				return
			}
		}

		auth := &AuthContext{
			KeyID:      id,
			KeyType:    keyType,
			Plan:       plan,
			IsLinkedIn: keyType == "linkedin",
		}

		ctx := context.WithValue(
			r.Context(),
			authContextKey,
			auth,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}

func parseTime(value string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}

	var lastErr error

	for _, format := range formats {
		t, err := time.Parse(format, value)
		if err == nil {
			return t, nil
		}
		lastErr = err
	}

	return time.Time{}, lastErr
}
