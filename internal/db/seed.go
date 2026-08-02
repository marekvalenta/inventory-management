package db

import (
	"database/sql"
	"log"

	"github.com/google/uuid"
)

func AutoSeed(database *sql.DB) {
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM locations").Scan(&count); err != nil {
		log.Fatalf("failed to check locations table: %v", err)
	}

	if count > 0 {
		return
	}

	rootID := uuid.New().String()

	if _, err := database.Exec(
		"INSERT INTO locations (id, name, description) VALUES (?, ?, ?)",
		rootID, "Home", "Root location",
	); err != nil {
		log.Fatalf("failed to seed root location: %v", err)
	}

	if _, err := database.Exec(
		"INSERT INTO settings (id, app_name, theme, root_location_id) VALUES (1, ?, ?, ?)",
		"Inventory", "system", rootID,
	); err != nil {
		log.Fatalf("failed to seed settings: %v", err)
	}

	log.Printf("auto-seeded root location %q and settings", "Home")
}
