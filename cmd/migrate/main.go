package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/turserg/go-service-template/internal/platform/config"
	"github.com/turserg/go-service-template/internal/platform/logger"
	postgresplatform "github.com/turserg/go-service-template/internal/platform/postgres"
)

func main() {
	cfg := config.Load()
	log := logger.NewJSON(cfg.ServiceName + "-migrator")

	if cfg.PostgresDSN == "" {
		log.Error("POSTGRES_DSN is required for migrations")
		os.Exit(1)
	}

	if err := run(cfg); err != nil {
		log.Error("migration failed", "error", err)
		os.Exit(1)
	}

	log.Info("migrations completed successfully", "dir", cfg.MigrationsDir)
}

func run(cfg config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := postgresplatform.ApplyMigrations(ctx, cfg.PostgresDSN, cfg.MigrationsDir); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}
