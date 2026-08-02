package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func Connect(dataDir string) *sql.DB {
	dsn := fmt.Sprintf("%s/inventory.db?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=5000", dataDir)

	database, err := sql.Open("sqlite", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	database.SetMaxOpenConns(4)
	database.SetMaxIdleConns(4)

	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		log.Fatalf("failed to enable foreign keys: %v", err)
	}

	return database
}

func RunMigrations(database *sql.DB) {
	goose.SetDialect("sqlite3")

	if err := goose.Up(database, "migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
}
