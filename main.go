package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"sktts-go/config"
	"sktts-go/database"
	"sktts-go/middleware"
	"sktts-go/models"
	"sktts-go/services"
)

const (
	piperPath  = "/usr/local/bin/piper"
	modelPath  = "/opt/xtts/model.onnx"
	voicesPath = "/opt/xtts/voices"
	serverName = "SKTTS-Go"
	version    = "1.1.0"
)

var supportedFormats = []string{
	"ogg",
	"opus",
	"mp3",
	"wav",
	"flac",
	"m4a",
	"webm",
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// ============================================================
// VOICES
// ============================================================

type voiceDefinition struct {
        Language string
        Name     string
        Model    string
}

var voiceDefinitions = []voiceDefinition{
        // PT-BR
        {"pt-BR", "Cadu", filepath.Join(voicesPath, "Cadu", "model.onnx")},
        {"pt-BR", "Dii", filepath.Join(voicesPath, "Dii", "model.onnx")},
        {"pt-BR", "Edresson", filepath.Join(voicesPath, "Edresson", "model.onnx")},
        {"pt-BR", "Faber", filepath.Join(voicesPath, "Faber", "model.onnx")},
        {"pt-BR", "Jeff", filepath.Join(voicesPath, "Jeff", "model.onnx")},
        {"pt-BR", "Miro", filepath.Join(voicesPath, "Miro", "model.onnx")},
        {"pt-BR", "Razo", filepath.Join(voicesPath, "Razo", "model.onnx")},
        {"pt-BR", "Wesley", filepath.Join(voicesPath, "Wesley", "model.onnx")},

        // EN-US
        {"en-US", "hfc_female", filepath.Join(voicesPath, "en_US", "hfc_female", "medium", "en_US-hfc_female-medium.onnx")},
        {"en-US", "hfc_male", filepath.Join(voicesPath, "en_US", "hfc_male", "medium", "en_US-hfc_male-medium.onnx")},
        {"en-US", "Joe", filepath.Join(voicesPath, "en_US", "joe", "medium", "en_US-joe-medium.onnx")},
        {"en-US", "John", filepath.Join(voicesPath, "en_US", "john", "medium", "en_US-john-medium.onnx")},

        // ES-ES
        {"es-ES", "CarlFM", filepath.Join(voicesPath, "es_ES", "carlfm", "x_low", "es_ES-carlfm-x_low.onnx")},
        {"es-ES", "Sharvard", filepath.Join(voicesPath, "es_ES", "sharvard", "medium", "es_ES-sharvard-medium.onnx")},
}

func listVoices() []string {
        voices := make([]string, 0, len(voiceDefinitions))

        for _, voice := range voiceDefinitions {
                if info, err := os.Stat(voice.Model); err == nil && !info.IsDir() {
                        voices = append(voices, voice.Name)
                }
        }

        sort.Strings(voices)

        return voices
}

func listVoicesByLanguage() map[string][]string {
        result := make(map[string][]string)

        for _, voice := range voiceDefinitions {
                if info, err := os.Stat(voice.Model); err != nil || info.IsDir() {
                        continue
                }

                result[voice.Language] = append(
                        result[voice.Language],
                        voice.Name,
                )
        }

        for language := range result {
                sort.Strings(result[language])
        }

        return result
}

func resolveVoiceModel(voice string, language string) (string, string, error) {

        voice = strings.TrimSpace(voice)
        language = strings.TrimSpace(language)

        // Sem voice:
        // mantém compatibilidade com o modelo original.
        if voice == "" {
                return modelPath, "default", nil
        }

        // Segurança contra path traversal.
        if filepath.Base(voice) != voice ||
                voice == "." ||
                voice == ".." {

                return "", "", fmt.Errorf("invalid voice")
        }

        if language != "" {

                if filepath.Base(language) != language ||
                        language == "." ||
                        language == ".." {

                        return "", "", fmt.Errorf("invalid language")
                }

                for _, definition := range voiceDefinitions {

                        if !strings.EqualFold(
                                definition.Language,
                                language,
                        ) || !strings.EqualFold(
                                definition.Name,
                                voice,
                        ) {
                                continue
                        }

                        info, err := os.Stat(definition.Model)

                        if err != nil || info.IsDir() {
                                return "", "", fmt.Errorf(
                                        "voice not found: %s",
                                        voice,
                                )
                        }

                        return definition.Model, definition.Name, nil
                }

                return "", "", fmt.Errorf(
                        "voice not found for language %s: %s",
                        language,
                        voice,
                )
        }

        // Compatibilidade com a API anterior:
        // voice=Wesley
        // voice=Cadu
        // etc.
        for _, definition := range voiceDefinitions {

                if !strings.EqualFold(
                        definition.Name,
                        voice,
                ) {
                        continue
                }

                info, err := os.Stat(definition.Model)

                if err != nil || info.IsDir() {
                        return "", "", fmt.Errorf(
                                "voice not found: %s",
                                voice,
                        )
                }

                return definition.Model, definition.Name, nil
        }

        return "", "", fmt.Errorf(
                "voice not found: %s",
                voice,
        )
}

// ============================================================
// API KEY
// ============================================================

// ============================================================
// ADMIN API KEYS
// ============================================================

type createKeyRequest struct {
	Name      string  `json:"name"`
	KeyType   string  `json:"key_type"`
	Plan      string  `json:"plan"`
	ExpiresAt *string `json:"expires_at,omitempty"`
}

func adminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authCtx := middleware.Get(r)

		if authCtx == nil || !authCtx.IsAdmin {
			writeJSON(w, http.StatusForbidden, map[string]any{
				"success": false,
				"error":   "acesso administrativo necessário",
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func createAdminKeyHandler(keyService *services.KeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var req createKeyRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"error":   "JSON inválido",
			})
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		req.KeyType = strings.TrimSpace(req.KeyType)
		req.Plan = strings.TrimSpace(req.Plan)

		if req.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"error":   "name é obrigatório",
			})
			return
		}

		if req.KeyType == "" {
			req.KeyType = "client"
		}

		if req.Plan == "" {
			req.Plan = "default"
		}

		var expiresAt *time.Time

		if req.ExpiresAt != nil && strings.TrimSpace(*req.ExpiresAt) != "" {
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.ExpiresAt))

			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{
					"success": false,
					"error":   "expires_at deve estar em formato RFC3339",
				})
				return
			}

			expiresAt = &t
		}

		item, rawKey, err := keyService.Create(
			req.Name,
			req.KeyType,
			req.Plan,
			expiresAt,
		)

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"success": true,
			"key":     rawKey,
			"data":    item,
			"warning": "guarde esta chave agora; ela não será armazenada em texto puro.",
		})
	}
}

func listAdminKeysHandler(keyService *services.KeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		keys, err := keyService.List()

		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		if keys == nil {
			keys = []models.APIKey{}
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"count":   len(keys),
			"data":    keys,
		})
	}
}

func getAdminKeyHandler(keyService *services.KeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := chi.URLParam(r, "id")

		item, err := keyService.Get(id)

		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{
				"success": false,
				"error":   "chave não encontrada",
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"data":    item,
		})
	}
}

func suspendAdminKeyHandler(keyService *services.KeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := chi.URLParam(r, "id")

		if err := keyService.Suspend(id); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "chave suspensa",
			"id":      id,
		})
	}
}

func activateAdminKeyHandler(keyService *services.KeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := chi.URLParam(r, "id")

		if err := keyService.Activate(id); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "chave ativada",
			"id":      id,
		})
	}
}

func revokeAdminKeyHandler(keyService *services.KeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := chi.URLParam(r, "id")

		if err := keyService.Revoke(id); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "chave revogada",
			"id":      id,
		})
	}
}

func deleteAdminKeyHandler(keyService *services.KeyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := chi.URLParam(r, "id")

		if err := keyService.Delete(id); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"message": "chave marcada como deletada",
			"id":      id,
		})
	}
}

// ============================================================
// MAIN
// ============================================================

func createAdminPlanHandler(plansService *services.PlansService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Name                 string  `json:"name"`
			Description          string  `json:"description"`
			Price                float64 `json:"price"`
			RequestsPerMinute    int     `json:"requests_per_minute"`
			RequestsPerDay       int     `json:"requests_per_day"`
			CharactersPerRequest int     `json:"characters_per_request"`
			CharactersPerDay     int     `json:"characters_per_day"`
			MaxConcurrentTTS     int     `json:"max_concurrent_tts"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, `{"error":"JSON inválido"}`, http.StatusBadRequest)
			return
		}

		plan, err := plansService.Create(
			input.Name,
			input.Description,
			input.Price,
			input.RequestsPerMinute,
			input.RequestsPerDay,
			input.CharactersPerRequest,
			input.CharactersPerDay,
			input.MaxConcurrentTTS,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(plan)
	}
}

func listAdminPlansHandler(plansService *services.PlansService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plans, err := plansService.List()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(plans)
	}
}

func getAdminPlanHandler(plansService *services.PlansService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		plan, err := plansService.Get(id)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(plan)
	}
}

func updateAdminPlanHandler(plansService *services.PlansService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		var input struct {
			Name                 string  `json:"name"`
			Description          string  `json:"description"`
			Price                float64 `json:"price"`
			RequestsPerMinute    int     `json:"requests_per_minute"`
			RequestsPerDay       int     `json:"requests_per_day"`
			CharactersPerRequest int     `json:"characters_per_request"`
			CharactersPerDay     int     `json:"characters_per_day"`
			MaxConcurrentTTS     int     `json:"max_concurrent_tts"`
			Status               string  `json:"status"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, `{"error":"JSON inválido"}`, http.StatusBadRequest)
			return
		}

		plan, err := plansService.Update(
			id,
			input.Name,
			input.Description,
			input.Price,
			input.RequestsPerMinute,
			input.RequestsPerDay,
			input.CharactersPerRequest,
			input.CharactersPerDay,
			input.MaxConcurrentTTS,
			input.Status,
		)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(plan)
	}
}

func deleteAdminPlanHandler(plansService *services.PlansService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		if err := plansService.Delete(id); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "deleted",
		})
	}
}

func main() {
	cfg := config.Load()

	if cfg.AdminKey == "" {
		fmt.Println("AVISO: SKTTS_ADMIN_KEY não configurada")
	}

	db, err := database.Open(cfg.DatabasePath)
	if err != nil {
		fmt.Println("Erro ao abrir banco:", err)
		return
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		fmt.Println("Erro na migration:", err)
		return
	}

	usageService := services.NewUsageService(db)
	ipBlockService := services.NewIPBlockService(db)
	plansService := services.NewPlansService(db)
	concurrencyService := services.NewConcurrencyService()

	quotaService := services.NewQuotaService(
		usageService,
		ipBlockService,
		plansService,
	)

	auth := &middleware.Auth{
		APIKey:          cfg.APIKey,
		AdminKey:        cfg.AdminKey,
		LinkedInFreeKey: cfg.LinkedInFreeKey,
		DB:              db.SQL,
	}

	keyService := services.NewKeyService(db)
	audioService := services.NewAudioService(db.SQL)
	_ = audioService

	_ = quotaService

	r := chi.NewRouter()

	// ========================================================
	// HEALTH
	// ========================================================

	r.Get("/docs", docsHandler)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {

		writeJSON(
			w,
			http.StatusOK,
			map[string]any{
				"status":  "healthy",
				"device":  "cpu",
				"name":    serverName,
				"version": version,
			},
		)
	})

	// ========================================================
	// INFO
	// ========================================================

	r.Get("/info", func(w http.ResponseWriter, r *http.Request) {

		writeJSON(
			w,
			http.StatusOK,
			map[string]any{
				"name":            serverName,
				"version":         version,
				"engine":          "Piper",
				"device":          "cpu",
				"format_required": true,
				"formats":         supportedFormats,
				"voices":          listVoices(),
				"voices_by_language": listVoicesByLanguage(),
				"languages":          []string{"pt-BR", "en-US", "es-ES"},
				"voice_optional":  true,
				"default_voice":   "default",
			},
		)
	})

	// ========================================================
	// TTS
	// ========================================================

	r.With(auth.Middleware).Post(
		"/tts",
		func(w http.ResponseWriter, r *http.Request) {

			// ------------------------------------------------
			// TEXTO
			// ------------------------------------------------

			text := strings.TrimSpace(
				r.FormValue("text"),
			)

			if text == "" {

				writeJSON(
					w,
					http.StatusBadRequest,
					map[string]any{
						"error": "text is required",
					},
				)

				return
			}

			// ------------------------------------------------
			// FORMAT
			// OBRIGATÓRIO
			// ------------------------------------------------

			format := strings.ToLower(
				strings.TrimSpace(
					r.FormValue("format"),
				),
			)

			if format == "" {

				writeJSON(
					w,
					http.StatusBadRequest,
					map[string]any{
						"error":             "format is required",
						"supported_formats": supportedFormats,
					},
				)

				return
			}

			// ------------------------------------------------
			// VALIDAR FORMAT
			// ------------------------------------------------

			switch format {

			case "ogg":
			case "opus":
			case "mp3":
			case "wav":
			case "flac":
			case "m4a":
			case "webm":

			default:

				writeJSON(
					w,
					http.StatusBadRequest,
					map[string]any{
						"error": fmt.Sprintf(
							"unsupported format: %s",
							format,
						),
						"supported_formats": supportedFormats,
					},
				)

				return
			}

			// ------------------------------------------------
			// VOICE
			//
			// Opcional.
			//
			// voice=Wesley
			// ------------------------------------------------

			language := strings.TrimSpace(
				r.FormValue("language"),
			)

			voice := strings.TrimSpace(
				r.FormValue("voice"),
			)

			selectedModel, selectedVoice, err :=
				resolveVoiceModel(voice, language)

			if err != nil {

				writeJSON(
					w,
					http.StatusBadRequest,
					map[string]any{
						"success": false,
						"error":   err.Error(),
						"voices":  listVoices(),
					},
				)

				return
			}

			// ------------------------------------------------
			// PIPER -> PCM RAW
			// ------------------------------------------------

			// ------------------------------------------------
			// QUOTA / IP
			// ------------------------------------------------

			authContext := middleware.Get(r)

			if authContext == nil {
				middleware.WriteJSON(
					w,
					http.StatusUnauthorized,
					map[string]any{
						"success": false,
						"error":   "authentication_context_missing",
					},
				)
				return
			}

			clientIP := r.RemoteAddr

			if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
				clientIP = host
			}

			status, quotaError, maxConcurrentTTS := quotaService.Check(
				authContext.KeyID,
				authContext.Plan,
				clientIP,
				len([]rune(text)),
			)

			if status != 0 {
				middleware.WriteJSON(
					w,
					status,
					map[string]any{
						"success": false,
						"error":   quotaError,
					},
				)
				return
			}

			// ------------------------------------------------
			// LIMITE GLOBAL DE CONCORRENCIA
			// ------------------------------------------------
			// No maximo 2 geracoes Piper/FFmpeg simultaneas.
			releaseConcurrency := concurrencyService.Acquire(
				authContext.Plan,
				maxConcurrentTTS,
			)
			defer releaseConcurrency()

			var pcm bytes.Buffer

			piper := exec.Command(
				piperPath,
				"--model",
				selectedModel,
				"--output-raw",
			)

			piper.Stdin = strings.NewReader(text)
			piper.Stdout = &pcm

			var piperErr bytes.Buffer
			piper.Stderr = &piperErr

			if err := piper.Run(); err != nil {

				http.Error(
					w,
					fmt.Sprintf(
						"piper error: %v",
						err,
					),
					http.StatusInternalServerError,
				)

				return
			}

			if pcm.Len() == 0 {

				http.Error(
					w,
					"piper produced no audio",
					http.StatusInternalServerError,
				)

				return
			}

			// ------------------------------------------------
			// FFMPEG
			// ------------------------------------------------

			var audio bytes.Buffer

			args := []string{
				"-y",

				"-f",
				"s16le",

				"-ar",
				"22050",

				"-ac",
				"1",

				"-i",
				"pipe:0",
			}

			switch format {

			// ------------------------------------------------
			// OGG / OPUS
			// ------------------------------------------------

			case "ogg", "opus":

				args = append(
					args,

					"-c:a",
					"libopus",

					"-b:a",
					"32k",

					"-vbr",
					"on",

					"-ar",
					"24000",

					"-ac",
					"1",

					"-f",
					"ogg",
				)

			// ------------------------------------------------
			// MP3
			// ------------------------------------------------

			case "mp3":

				args = append(
					args,

					"-c:a",
					"libmp3lame",

					"-b:a",
					"64k",

					"-ar",
					"22050",

					"-ac",
					"1",

					"-f",
					"mp3",
				)

			// ------------------------------------------------
			// WAV
			// ------------------------------------------------

			case "wav":

				args = append(
					args,

					"-c:a",
					"pcm_s16le",

					"-ar",
					"22050",

					"-ac",
					"1",

					"-f",
					"wav",
				)

			// ------------------------------------------------
			// FLAC
			// ------------------------------------------------

			case "flac":

				args = append(
					args,

					"-c:a",
					"flac",

					"-ar",
					"22050",

					"-ac",
					"1",

					"-f",
					"flac",
				)

			// ------------------------------------------------
			// M4A / AAC
			// ------------------------------------------------

			case "m4a":

				args = append(
					args,

					"-c:a",
					"aac",

					"-b:a",
					"96k",

					"-ar",
					"22050",

					"-ac",
					"1",

					"-movflags",
					"+faststart",

					"-f",
					"ipod",
				)

			// ------------------------------------------------
			// WEBM / OPUS
			// ------------------------------------------------

			case "webm":

				args = append(
					args,

					"-c:a",
					"libopus",

					"-b:a",
					"32k",

					"-vbr",
					"on",

					"-ar",
					"24000",

					"-ac",
					"1",

					"-f",
					"webm",
				)
			}

			// ------------------------------------------------
			// M4A
			//
			// Precisa de arquivo seekable.
			// ------------------------------------------------

			if format == "m4a" {

				tempFile :=
					"/tmp/sktss-" +
						uuid.New().String() +
						".m4a"

				defer os.Remove(tempFile)

				args = append(
					args,
					tempFile,
				)

				ffmpeg := exec.Command(
					"ffmpeg",
					args...,
				)

				ffmpeg.Stdin = &pcm

				var ffmpegErr bytes.Buffer
				ffmpeg.Stderr = &ffmpegErr

				if err := ffmpeg.Run(); err != nil {

					http.Error(
						w,
						fmt.Sprintf(
							"ffmpeg error: %v - %s",
							err,
							ffmpegErr.String(),
						),
						http.StatusInternalServerError,
					)

					return
				}

				data, err := os.ReadFile(tempFile)

				if err != nil {

					http.Error(
						w,
						fmt.Sprintf(
							"erro ao ler M4A: %v",
							err,
						),
						http.StatusInternalServerError,
					)

					return
				}

				audio.Write(data)

			} else {

				args = append(
					args,
					"pipe:1",
				)

				ffmpeg := exec.Command(
					"ffmpeg",
					args...,
				)

				ffmpeg.Stdin = &pcm
				ffmpeg.Stdout = &audio

				var ffmpegErr bytes.Buffer
				ffmpeg.Stderr = &ffmpegErr

				if err := ffmpeg.Run(); err != nil {

					http.Error(
						w,
						fmt.Sprintf(
							"ffmpeg error: %v",
							err,
						),
						http.StatusInternalServerError,
					)

					return
				}
			}

			if audio.Len() == 0 {

				http.Error(
					w,
					"ffmpeg produced no audio",
					http.StatusInternalServerError,
				)

				return
			}

			// ------------------------------------------------
			// REGISTRAR USO
			// ------------------------------------------------

			if authContext != nil && !authContext.IsAdmin {
				if err := usageService.Add(
					authContext.KeyID,
					clientIP,
					len([]rune(text)),
				); err != nil {
					fmt.Println("Aviso: erro ao registrar uso:", err)
				}
			}

			// ------------------------------------------------
			// VOICE HEADERS
			// ------------------------------------------------

			w.Header().Set(
				"X-Voice",
				selectedVoice,
			)

			w.Header().Set(
				"X-Voice-Model",
				selectedModel,
			)

			// ------------------------------------------------
			// RESPONSE HEADERS
			// ------------------------------------------------

			var contentType string
			var extension string

			switch format {

			case "ogg", "opus":

				contentType = "audio/ogg; codecs=opus"
				extension = ".ogg"

				w.Header().Set(
					"X-Audio-Codec",
					"opus",
				)

				w.Header().Set(
					"X-Audio-Container",
					"ogg",
				)

				w.Header().Set(
					"X-Audio-Sample-Rate",
					"24000",
				)

				w.Header().Set(
					"X-Audio-Channels",
					"1",
				)

			case "mp3":

				contentType = "audio/mpeg"
				extension = ".mp3"

				w.Header().Set(
					"X-Audio-Codec",
					"mp3",
				)

				w.Header().Set(
					"X-Audio-Sample-Rate",
					"22050",
				)

				w.Header().Set(
					"X-Audio-Channels",
					"1",
				)

			case "wav":

				contentType = "audio/wav"
				extension = ".wav"

				w.Header().Set(
					"X-Audio-Codec",
					"pcm_s16le",
				)

				w.Header().Set(
					"X-Audio-Sample-Rate",
					"22050",
				)

				w.Header().Set(
					"X-Audio-Channels",
					"1",
				)

			case "flac":

				contentType = "audio/flac"
				extension = ".flac"

				w.Header().Set(
					"X-Audio-Codec",
					"flac",
				)

				w.Header().Set(
					"X-Audio-Sample-Rate",
					"22050",
				)

				w.Header().Set(
					"X-Audio-Channels",
					"1",
				)

			case "m4a":

				contentType = "audio/mp4"
				extension = ".m4a"

				w.Header().Set(
					"X-Audio-Codec",
					"aac",
				)

				w.Header().Set(
					"X-Audio-Container",
					"m4a",
				)

				w.Header().Set(
					"X-Audio-Sample-Rate",
					"22050",
				)

				w.Header().Set(
					"X-Audio-Channels",
					"1",
				)

			case "webm":

				contentType = "audio/webm"
				extension = ".webm"

				w.Header().Set(
					"X-Audio-Codec",
					"opus",
				)

				w.Header().Set(
					"X-Audio-Container",
					"webm",
				)

				w.Header().Set(
					"X-Audio-Sample-Rate",
					"24000",
				)

				w.Header().Set(
					"X-Audio-Channels",
					"1",
				)
			}

			// ------------------------------------------------
			// RESPONSE
			// ------------------------------------------------

			filename :=
				uuid.New().String() +
					extension

			w.Header().Set(
				"Content-Type",
				contentType,
			)

			w.Header().Set(
				"Content-Length",
				fmt.Sprintf(
					"%d",
					audio.Len(),
				),
			)

			w.Header().Set(
				"Content-Disposition",
				fmt.Sprintf(
					"inline; filename=%s",
					filename,
				),
			)

			// -------------------------------------------
			// SALVAR AUDIO NO HISTORICO
			// -------------------------------------------

			if authContext != nil && !authContext.IsAdmin {
				audioFilename := uuid.New().String() + extension

				if _, err := audioService.Create(
					authContext.KeyID,
					audioFilename,
					selectedVoice,
					format,
					contentType,
					len([]rune(text)),
					audio.Bytes(),
				); err != nil {
					fmt.Println("Aviso: erro ao salvar histórico de áudio:", err)
				}
			}

			_, _ = w.Write(
				audio.Bytes(),
			)
		},
	)

	// ========================================================
	// ========================================================
	// ADMIN API KEYS
	// ========================================================

	r.With(auth.Middleware, adminOnly).Post(
		"/admin/keys",
		createAdminKeyHandler(keyService),
	)

	r.With(auth.Middleware, adminOnly).Get(
		"/admin/keys",
		listAdminKeysHandler(keyService),
	)

	r.With(auth.Middleware, adminOnly).Get(
		"/admin/keys/{id}",
		getAdminKeyHandler(keyService),
	)

	r.With(auth.Middleware, adminOnly).Post(
		"/admin/keys/{id}/suspend",
		suspendAdminKeyHandler(keyService),
	)

	r.With(auth.Middleware, adminOnly).Post(
		"/admin/keys/{id}/activate",
		activateAdminKeyHandler(keyService),
	)

	r.With(auth.Middleware, adminOnly).Post(
		"/admin/keys/{id}/revoke",
		revokeAdminKeyHandler(keyService),
	)

	r.With(auth.Middleware, adminOnly).Delete(
		"/admin/keys/{id}",
		deleteAdminKeyHandler(keyService),
	)

	// ========================================================
	// ADMIN PLANS
	// ========================================================

	r.With(auth.Middleware, adminOnly).Post(
		"/admin/plans",
		createAdminPlanHandler(plansService),
	)

	r.With(auth.Middleware, adminOnly).Get(
		"/admin/plans",
		listAdminPlansHandler(plansService),
	)

	r.With(auth.Middleware, adminOnly).Get(
		"/admin/plans/{id}",
		getAdminPlanHandler(plansService),
	)

	r.With(auth.Middleware, adminOnly).Put(
		"/admin/plans/{id}",
		updateAdminPlanHandler(plansService),
	)

	r.With(auth.Middleware, adminOnly).Delete(
		"/admin/plans/{id}",
		deleteAdminPlanHandler(plansService),
	)

	// SERVER
	// ========================================================

	fmt.Println(
		"SKTTS-Go iniciado em",
		cfg.Address,
	)

	if err := http.ListenAndServe(
		cfg.Address,
		r,
	); err != nil {

		fmt.Println(
			"Erro no servidor:",
			err,
		)
	}
}
