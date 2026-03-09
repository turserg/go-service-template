package postgres

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func ApplyMigrations(ctx context.Context, dsn, dir string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open sql connection for migrations: %w", err)
	}
	defer db.Close()

	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping sql connection for migrations: %w", err)
	}

	if err = goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err = goose.UpContext(ctx, db, dir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}
