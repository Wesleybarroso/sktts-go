package config

import "os"

type Config struct {
	APIKey          string
	AdminKey        string
	LinkedInFreeKey string
	DatabasePath    string
	Address         string

	MaxConcurrentTTS int
	MaxTextChars     int
}

func Load() Config {
	return Config{
		APIKey:          os.Getenv("XTTS_API_KEY"),
		AdminKey:        os.Getenv("SKTTS_ADMIN_KEY"),
		LinkedInFreeKey: os.Getenv("SKTTS_LINKEDIN_FREE_KEY"),

		DatabasePath: getenv(
			"SKTTS_DB",
			"/opt/xtts/data/sktss.db",
		),

		Address: getenv(
			"SKTTS_ADDR",
			":8000",
		),

		MaxConcurrentTTS: 2,
		MaxTextChars:     5000,
	}
}

func getenv(name, fallback string) string {
	value := os.Getenv(name)

	if value == "" {
		return fallback
	}

	return value
}
