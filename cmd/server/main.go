package main

import (
	"embed"
	"log"
	"net/http"

	"github.com/marekvalenta/inventory-management/internal/config"
	"github.com/marekvalenta/inventory-management/internal/db"
	"github.com/marekvalenta/inventory-management/internal/router"
)

//go:embed all:static
var embeddedFrontend embed.FS

func main() {
	cfg := config.Load()

	database := db.Connect(cfg.DataDir)
	defer database.Close()

	db.RunMigrations(database)
	db.AutoSeed(database)

	r := router.New(embeddedFrontend, database)

	log.Printf("inventory-management server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
