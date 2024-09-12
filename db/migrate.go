package db

import (
	"context"
	"embed"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib" //nolint:revive
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" //nolint:revive
)

//go:embed migrations/*
var migrationsFS embed.FS

func MigrateUp(ctx context.Context, dbURL string) error {
	return runMigrations(ctx, "up", dbURL)
}

func MigrateDown(ctx context.Context, dbURL string) error {
	return runMigrations(ctx, "down", dbURL)
}

func runMigrations(ctx context.Context, cmd, dbURL string) error {
	db, err := goose.OpenDBWithDriver(getDriver(dbURL), dbURL)
	if err != nil {
		return err
	}

	defer func() {
		_ = db.Close()
	}()

	goose.SetBaseFS(migrationsFS)
	if err := goose.RunContext(ctx, cmd, db, "migrations"); err != nil {
		return err
	}

	return nil
}

func getDriver(dbURL string) string {
	driver := "sqlite"

	if strings.HasPrefix(dbURL, "postgres") {
		driver = "pgx"
	}

	return driver
}
