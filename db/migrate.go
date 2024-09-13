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
	driver, directory := getDBConfig(dbURL)
	db, err := goose.OpenDBWithDriver(driver, dbURL)
	if err != nil {
		return err
	}

	defer func() {
		_ = db.Close()
	}()

	goose.SetBaseFS(migrationsFS)
	if err := goose.RunContext(ctx, cmd, db, directory); err != nil {
		return err
	}

	return nil
}

func getDBConfig(dbURL string) (driver, directory string) {
	driver = "sqlite"
	directory = "migrations/sqlite"

	if strings.HasPrefix(dbURL, "postgres") {
		driver = "pgx"
		directory = "migrations/postgres"
	}

	return driver, directory
}
