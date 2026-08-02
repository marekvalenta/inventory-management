package router

import (
	"database/sql"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/marekvalenta/inventory-management/internal/handler"
	"github.com/marekvalenta/inventory-management/internal/service"
)

func New(embeddedFrontend fs.FS, db *sql.DB) chi.Router {
	distFS, err := fs.Sub(embeddedFrontend, "static")
	if err != nil {
		log.Printf("embedded static directory not found: %v", err)
		distFS = nil
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	locationSvc := service.NewLocationService(db)
	locationHandler := handler.NewLocationHandler(locationSvc)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", handler.HealthHandler())
		locationHandler.RegisterRoutes(r)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if isAPIRequest(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found","code":"not_found"}`))
			return
		}
		serveSPA(w, r, distFS)
	})

	return r
}

func NewTestRouter(db *sql.DB) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	locationSvc := service.NewLocationService(db)
	locationHandler := handler.NewLocationHandler(locationSvc)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", handler.HealthHandler())
		locationHandler.RegisterRoutes(r)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found","code":"not_found"}`))
	})

	return r
}

func isAPIRequest(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/")
}

func serveSPA(w http.ResponseWriter, r *http.Request, distFS fs.FS) {
	if distFS == nil {
		http.Error(w, "frontend not available", http.StatusServiceUnavailable)
		return
	}

	filePath := strings.TrimPrefix(r.URL.Path, "/")
	if filePath == "" {
		filePath = "index.html"
	}

	f, err := distFS.Open(filePath)
	if err == nil {
		f.Close()
		http.FileServer(http.FS(distFS)).ServeHTTP(w, r)
		return
	}

	indexFile, err := distFS.Open("index.html")
	if err != nil {
		log.Printf("index.html not found in embedded frontend: %v", err)
		http.Error(w, "frontend not available", http.StatusServiceUnavailable)
		return
	}
	indexFile.Close()

	http.ServeFileFS(w, r, distFS, "index.html")
}
