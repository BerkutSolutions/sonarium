package migrations

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/pressly/goose/v3"
)

const (
	defaultLocalDir  = "internal/storage/migrations"
	defaultDockerDir = "/app/migrations"
)

func ResolveDir() string {
	if envDir := os.Getenv("MIGRATIONS_DIR"); envDir != "" {
		return envDir
	}

	if _, err := os.Stat(defaultDockerDir); err == nil {
		return defaultDockerDir
	}

	return defaultLocalDir
}

func Up(db *sql.DB, dir string) error {
	if dir == "" {
		dir = ResolveDir()
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Up(db, dir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}

func Down(db *sql.DB, dir string) error {
	if dir == "" {
		dir = ResolveDir()
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.Down(db, dir); err != nil {
		return fmt.Errorf("goose down: %w", err)
	}

	return nil
}
