package handler

import (
	"net/http"

	"tournament-games/internal/db"
	"tournament-games/internal/model"
)

type LeaderboardPageData struct {
	BaseData
	Entries []*model.LeaderboardEntry
}

func (a *App) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := db.GetLeaderboard(a.DB)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	a.Tmpl.Page(w, "leaderboard", LeaderboardPageData{
		BaseData: a.baseData(r),
		Entries:  entries,
	})
}

func (a *App) handleLeaderboardPartial(w http.ResponseWriter, r *http.Request) {
	entries, err := db.GetLeaderboard(a.DB)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	a.Tmpl.Partial(w, "leaderboard_rows", entries)
}
