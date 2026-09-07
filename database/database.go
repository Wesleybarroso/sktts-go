package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	SQL *sql.DB
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("criar diretório do banco: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("abrir banco: %w", err)
	}

	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("ativar WAL: %w", err)
	}

	if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("configurar busy_timeout: %w", err)
	}

	return &DB{SQL: db}, nil
}

func (db *DB) Close() error {
	if db == nil || db.SQL == nil {
		return nil
	}

	return db.SQL.Close()
}

func (db *DB) Migrate() error {
	_, err := db.SQL.Exec(`
CREATE TABLE IF NOT EXISTS api_keys (
	id TEXT PRIMARY KEY,
	key_hash TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	key_type TEXT NOT NULL,
	plan TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'active',
	expires_at DATETIME,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	revoked_at DATETIME,
	suspended_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_api_keys_hash
	ON api_keys(key_hash);

CREATE INDEX IF NOT EXISTS idx_api_keys_status
	ON api_keys(status);

CREATE TABLE IF NOT EXISTS plans (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL UNIQUE,
        description TEXT NOT NULL DEFAULT '',
        price REAL NOT NULL DEFAULT 0,
        requests_per_minute INTEGER NOT NULL DEFAULT 0,
        requests_per_day INTEGER NOT NULL DEFAULT 0,
        characters_per_request INTEGER NOT NULL DEFAULT 0,
        characters_per_day INTEGER NOT NULL DEFAULT 0,
        max_concurrent_tts INTEGER NOT NULL DEFAULT 1,
        status TEXT NOT NULL DEFAULT 'active',
        created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_plans_status
        ON plans(status);

INSERT OR IGNORE INTO plans
        (id, name, description, price, requests_per_minute,
         requests_per_day, characters_per_request, characters_per_day,
         max_concurrent_tts, status)
VALUES
        ('admin', 'Admin', 'Acesso administrativo sem limites', 0, 0, 0, 0, 0, 0, 'active'),
        ('free_24h', 'Free 24h', 'Plano gratuito de 24 horas', 0, 5, 20, 5000, 10000, 2, 'active'),
        ('free_30d', 'Free 30d', 'Plano gratuito de 30 dias', 0, 5, 50, 5000, 25000, 2, 'active'),
        ('linkedin', 'LinkedIn', 'Plano especial LinkedIn', 0, 5, 20, 5000, 10000, 2, 'active');

CREATE TABLE IF NOT EXISTS usage_daily (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	key_id TEXT NOT NULL,
	ip TEXT NOT NULL,
	usage_date TEXT NOT NULL,
	requests INTEGER NOT NULL DEFAULT 0,
	characters INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(key_id, ip, usage_date)
);

CREATE INDEX IF NOT EXISTS idx_usage_daily_key
	ON usage_daily(key_id);

CREATE INDEX IF NOT EXISTS idx_usage_daily_ip
	ON usage_daily(ip);

CREATE TABLE IF NOT EXISTS ip_blocks (
	ip TEXT PRIMARY KEY,
	reason TEXT NOT NULL,
	blocked_until DATETIME NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ip_blocks_until
	ON ip_blocks(blocked_until);



CREATE TABLE IF NOT EXISTS audio_files (
	id TEXT PRIMARY KEY,
	key_id TEXT NOT NULL,
	filename TEXT NOT NULL,
	voice TEXT NOT NULL,
	format TEXT NOT NULL,
	content_type TEXT NOT NULL,
	characters INTEGER NOT NULL DEFAULT 0,
	size_bytes INTEGER NOT NULL DEFAULT 0,
	path TEXT NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audio_files_key_id
	ON audio_files(key_id);

CREATE INDEX IF NOT EXISTS idx_audio_files_created_at
	ON audio_files(created_at);
`)

	if err != nil {
		return fmt.Errorf("migrar banco: %w", err)
	}

	return nil
}
