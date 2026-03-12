package main

import (
	"fmt"
	"os"

	"music-server/internal/platform/config"
	"music-server/internal/storage/migrations"
	"music-server/internal/storage/postgres"
)

func main() {
	direction := "up"
	if len(os.Args) > 1 {
		direction = os.Args[1]
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	db, err := postgres.Open(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open postgres: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = db.Close()
	}()

	switch direction {
	case "up":
		err = migrations.Up(db, migrations.ResolveDir())
	case "down":
		err = migrations.Down(db, migrations.ResolveDir())
	default:
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/migrate [up|down]")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "migration %s failed: %v\n", direction, err)
		os.Exit(1)
	}
}
