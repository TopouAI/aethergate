package main

import (
	"context"
	"fmt"
	"os"
	"time"

	postgresstorage "github.com/topoai/aethergate/apps/api/internal/storage/postgres"
)

func main() {
	databaseURL := os.Getenv("AETHERGATE_DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "AETHERGATE_DATABASE_URL is required")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	repository, err := postgresstorage.Open(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to postgres: %v\n", err)
		os.Exit(1)
	}
	defer repository.Close()
	if err := repository.Migrate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "apply migrations: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("AetherGate database is up to date")
}
