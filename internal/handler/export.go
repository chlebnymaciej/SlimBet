package handler

import (
	"fmt"
	"net/http"
	"time"

	"tournament-games/internal/export"
)

func (a *App) handleFixturesCSV(w http.ResponseWriter, r *http.Request) {
	filename := fmt.Sprintf("fixtures-%s.csv", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	if err := export.BuildFixturesCSV(a.DB, w); err != nil {
		http.Error(w, "export error", http.StatusInternalServerError)
	}
}

func (a *App) handleGroupsCSV(w http.ResponseWriter, r *http.Request) {
	filename := fmt.Sprintf("groups-%s.csv", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	if err := export.BuildGroupsCSV(a.DB, w); err != nil {
		http.Error(w, "export error", http.StatusInternalServerError)
	}
}

func (a *App) handleTournamentCSV(w http.ResponseWriter, r *http.Request) {
	filename := fmt.Sprintf("tournament-%s.csv", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	if err := export.BuildTournamentCSV(a.DB, a.Cfg, w); err != nil {
		http.Error(w, "export error", http.StatusInternalServerError)
	}
}
