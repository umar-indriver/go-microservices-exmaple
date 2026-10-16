package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"github.com/microservices/db-service/internal/config"
)

func InitDB(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := migrateSchema(db); err != nil {
		return nil, fmt.Errorf("failed to run database schema migrations: %w", err)
	}

	log.Println("[DB-Service] Successfully connected to PostgreSQL database & verified schema.")
	return db, nil
}

func migrateSchema(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		name VARCHAR(255) NOT NULL,
		role VARCHAR(50) NOT NULL DEFAULT 'user',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id UUID PRIMARY KEY,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		token_hash VARCHAR(255) UNIQUE NOT NULL,
		expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
		revoked BOOLEAN NOT NULL DEFAULT FALSE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);
	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
	`
	_, err := db.Exec(query)
	return err
}
