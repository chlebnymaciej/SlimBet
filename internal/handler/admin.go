package handler

import (
	"net/http"
	"strconv"
	"time"

	"tournament-games/internal/db"
	"tournament-games/internal/setup"
)

// ScorerInterface lets admin handler trigger scoring without importing cron.
type ScorerInterface interface {
	ScoreAll()
	ScoreOne(fixtureID int64) error
	FetchResultsNow()
}

type AdminPageData struct {
	BaseData
	FixtureCount int
	PointsExact  int
	PointsOutcome int
	PointsGroup  int
	Deadline     time.Time
	Flash        string
	Error        string
}

// SetScorer wires the cron scorer for admin actions.
func (a *App) SetScorer(s ScorerInterface) {
	a.scorer = s
}

func (a *App) handleAdminGet(w http.ResponseWriter, r *http.Request) {
	count, _ := db.FixtureCount(a.DB)
	a.Tmpl.Page(w, "admin", AdminPageData{
		BaseData:      a.baseData(r),
		FixtureCount:  count,
		PointsExact:   a.Cfg.PointsExact,
		PointsOutcome: a.Cfg.PointsOutcome,
		PointsGroup:   a.Cfg.PointsGroupWinner,
		Deadline:      a.Cfg.TournamentBetDeadline,
	})
}

func (a *App) handleAdminSetup(w http.ResponseWriter, r *http.Request) {
	if err := setup.PrefetchFixtures(a.DB, a.API, a.Cfg.CompetitionCode, true); err != nil {
		count, _ := db.FixtureCount(a.DB)
		a.Tmpl.Page(w, "admin", AdminPageData{
			BaseData:     a.baseData(r),
			FixtureCount: count,
			Error:        "Fetch failed: " + err.Error(),
		})
		return
	}
	count, _ := db.FixtureCount(a.DB)
	a.Tmpl.Page(w, "admin", AdminPageData{
		BaseData:     a.baseData(r),
		FixtureCount: count,
		Flash:        "Fixtures refreshed successfully.",
	})
}

func (a *App) handleAdminScoreAll(w http.ResponseWriter, r *http.Request) {
	if a.scorer != nil {
		a.scorer.ScoreAll()
	}
	count, _ := db.FixtureCount(a.DB)
	a.Tmpl.Page(w, "admin", AdminPageData{
		BaseData:      a.baseData(r),
		FixtureCount:  count,
		PointsExact:   a.Cfg.PointsExact,
		PointsOutcome: a.Cfg.PointsOutcome,
		PointsGroup:   a.Cfg.PointsGroupWinner,
		Deadline:      a.Cfg.TournamentBetDeadline,
		Flash:         "Scoring run complete.",
	})
}

func (a *App) handleAdminFetchResults(w http.ResponseWriter, r *http.Request) {
	if a.scorer != nil {
		a.scorer.FetchResultsNow()
	}
	count, _ := db.FixtureCount(a.DB)
	a.Tmpl.Page(w, "admin", AdminPageData{
		BaseData:      a.baseData(r),
		FixtureCount:  count,
		PointsExact:   a.Cfg.PointsExact,
		PointsOutcome: a.Cfg.PointsOutcome,
		PointsGroup:   a.Cfg.PointsGroupWinner,
		Deadline:      a.Cfg.TournamentBetDeadline,
		Flash:         "Fetched results from API and scored all finished matches.",
	})
}

func (a *App) handleAdminScoreOne(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if a.scorer != nil {
		if err := a.scorer.ScoreOne(id); err != nil {
			count, _ := db.FixtureCount(a.DB)
			a.Tmpl.Page(w, "admin", AdminPageData{
				BaseData:     a.baseData(r),
				FixtureCount: count,
				Error:        "Score error: " + err.Error(),
			})
			return
		}
	}
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (a *App) handleAdminConfig(w http.ResponseWriter, r *http.Request) {
	exact, _ := strconv.Atoi(r.FormValue("points_exact"))
	outcome, _ := strconv.Atoi(r.FormValue("points_outcome"))
	group, _ := strconv.Atoi(r.FormValue("points_group"))
	deadlineStr := r.FormValue("tournament_deadline")

	if exact < 0 || outcome < 0 || group < 0 {
		count, _ := db.FixtureCount(a.DB)
		a.Tmpl.Page(w, "admin", AdminPageData{
			BaseData:     a.baseData(r),
			FixtureCount: count,
			Error:        "Points must be non-negative.",
		})
		return
	}

	deadline := a.Cfg.TournamentBetDeadline
	if deadlineStr != "" {
		if t, err := time.Parse("2006-01-02T15:04", deadlineStr); err == nil {
			deadline = t.UTC()
		}
	}

	a.Cfg.Update(exact, outcome, group, deadline)
	_ = a.Cfg.Save()

	count, _ := db.FixtureCount(a.DB)
	a.Tmpl.Page(w, "admin", AdminPageData{
		BaseData:      a.baseData(r),
		FixtureCount:  count,
		PointsExact:   exact,
		PointsOutcome: outcome,
		PointsGroup:   group,
		Deadline:      deadline,
		Flash:         "Config saved.",
	})
}
