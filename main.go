package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"

	footballapi "tournament-games/internal/api"
	"tournament-games/internal/auth"
	"tournament-games/internal/config"
	"tournament-games/internal/cron"
	"tournament-games/internal/db"
	"tournament-games/internal/handler"
	"tournament-games/internal/setup"
)

//go:embed web
var webFS embed.FS

//go:embed migrations
var migrationsFS embed.FS

func main() {
	cfg, err := config.Load("appsettings.json")
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	database, err := db.Open(cfg.DBPath, migrationsFS)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	// Ensure admin user exists if credentials are configured.
	if cfg.AdminUsername != "" && cfg.AdminPassword != "" {
		hash, err := auth.HashPassword(cfg.AdminPassword)
		if err != nil {
			log.Fatalf("hash admin password: %v", err)
		}
		if err := db.EnsureAdmin(database, cfg.AdminUsername, hash); err != nil {
			log.Printf("ensure admin: %v", err)
		}
	}

	apiClient := footballapi.New(cfg.APIKey)

	if cfg.APIKey != "" {
		if err := setup.PrefetchFixtures(database, apiClient, cfg.LeagueID, cfg.Season, false); err != nil {
			log.Printf("setup: prefetch fixtures: %v", err)
		}
	} else {
		log.Println("WARNING: API_KEY not set — skipping fixture prefetch.")
	}

	sm := scs.New()
	sm.Store = db.NewSessionStore(database)
	sm.Lifetime = 7 * 24 * time.Hour
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Secure = false

	tmpl, err := handler.LoadTemplates(webFS)
	if err != nil {
		log.Fatalf("templates: %v", err)
	}

	app := &handler.App{
		DB:   database,
		SM:   sm,
		Cfg:  cfg,
		Tmpl: tmpl,
		API:  apiClient,
	}

	sc := cron.NewScorer(database, apiClient, cfg)
	sc.Start()
	defer sc.Stop()
	app.SetScorer(sc)

	mux := http.NewServeMux()
	app.RegisterRoutes(mux, webFS)

	h := sm.LoadAndSave(mux)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, h); err != nil {
		log.Fatal(err)
	}
}
