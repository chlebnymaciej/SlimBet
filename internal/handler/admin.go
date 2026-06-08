package handler

import (
	"fmt"
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
	FixtureCount  int
	PointsExact   int
	PointsOutcome int
	PointsGroup   int
	Points1st     int
	Points2nd     int
	Points3rd     int
	PointsScorer  int
	Deadline      time.Time
	ScorerCount   int
	TResults      db.TournamentResults
	Flash         string
	Error         string
}

// SetScorer wires the cron scorer for admin actions.
func (a *App) SetScorer(s ScorerInterface) {
	a.scorer = s
}

func (a *App) adminPageData(r *http.Request, flash, errMsg string) AdminPageData {
	count, _ := db.FixtureCount(a.DB)
	scorerCount, _ := db.GetScorerCount(a.DB)
	tResults, _ := db.GetTournamentResults(a.DB)
	pts1st, pts2nd, pts3rd, ptsScorer := a.Cfg.GetTournamentPoints()
	return AdminPageData{
		BaseData:      a.baseData(r),
		FixtureCount:  count,
		PointsExact:   a.Cfg.PointsExact,
		PointsOutcome: a.Cfg.PointsOutcome,
		PointsGroup:   a.Cfg.PointsGroupWinner,
		Points1st:     pts1st,
		Points2nd:     pts2nd,
		Points3rd:     pts3rd,
		PointsScorer:  ptsScorer,
		Deadline:      a.Cfg.TournamentBetDeadline,
		ScorerCount:   scorerCount,
		TResults:      tResults,
		Flash:         flash,
		Error:         errMsg,
	}
}

func (a *App) handleAdminGet(w http.ResponseWriter, r *http.Request) {
	a.Tmpl.Page(w, "admin", a.adminPageData(r, "", ""))
}

func (a *App) handleAdminSetup(w http.ResponseWriter, r *http.Request) {
	if err := setup.PrefetchFixtures(a.DB, a.API, a.Cfg.CompetitionCode, true); err != nil {
		a.Tmpl.Page(w, "admin", a.adminPageData(r, "", "Fetch failed: "+err.Error()))
		return
	}
	a.Tmpl.Page(w, "admin", a.adminPageData(r, "Fixtures refreshed successfully.", ""))
}

func (a *App) handleAdminScoreAll(w http.ResponseWriter, r *http.Request) {
	if a.scorer != nil {
		a.scorer.ScoreAll()
	}
	a.Tmpl.Page(w, "admin", a.adminPageData(r, "Scoring run complete.", ""))
}

func (a *App) handleAdminFetchResults(w http.ResponseWriter, r *http.Request) {
	if a.scorer != nil {
		a.scorer.FetchResultsNow()
	}
	a.Tmpl.Page(w, "admin", a.adminPageData(r, "Fetched results from API and scored all finished matches.", ""))
}

func (a *App) handleAdminScoreOne(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if a.scorer != nil {
		if err := a.scorer.ScoreOne(id); err != nil {
			a.Tmpl.Page(w, "admin", a.adminPageData(r, "", "Score error: "+err.Error()))
			return
		}
	}
	http.Redirect(w, r, a.BasePath+"/admin", http.StatusSeeOther)
}

func (a *App) handleAdminRefreshScorers(w http.ResponseWriter, r *http.Request) {
	items, err := a.API.FetchScorers(a.Cfg.CompetitionCode, 50)
	if err != nil {
		a.Tmpl.Page(w, "admin", a.adminPageData(r, "", "Fetch scorers failed: "+err.Error()))
		return
	}
	if err := db.RefreshScorerCandidates(a.DB, items); err != nil {
		a.Tmpl.Page(w, "admin", a.adminPageData(r, "", "Save scorers failed: "+err.Error()))
		return
	}
	a.Tmpl.Page(w, "admin", a.adminPageData(r, fmt.Sprintf("Loaded %d scorer candidates.", len(items)), ""))
}

func (a *App) handleAdminScoreTournament(w http.ResponseWriter, r *http.Request) {
	results := db.TournamentResults{
		Champion:   r.FormValue("champion"),
		RunnerUp:   r.FormValue("runner_up"),
		ThirdPlace: r.FormValue("third_place"),
		TopScorer:  r.FormValue("top_scorer"),
	}
	pts1st, pts2nd, pts3rd, ptsScorer := a.Cfg.GetTournamentPoints()
	n, err := db.SaveAndScoreTournament(a.DB, results, pts1st, pts2nd, pts3rd, ptsScorer)
	if err != nil {
		a.Tmpl.Page(w, "admin", a.adminPageData(r, "", "Scoring failed: "+err.Error()))
		return
	}
	a.Tmpl.Page(w, "admin", a.adminPageData(r, fmt.Sprintf("Tournament scored: %d users updated.", n), ""))
}

func (a *App) handleAdminConfig(w http.ResponseWriter, r *http.Request) {
	exact, _ := strconv.Atoi(r.FormValue("points_exact"))
	outcome, _ := strconv.Atoi(r.FormValue("points_outcome"))
	group, _ := strconv.Atoi(r.FormValue("points_group"))
	pts1st, _ := strconv.Atoi(r.FormValue("points_1st"))
	pts2nd, _ := strconv.Atoi(r.FormValue("points_2nd"))
	pts3rd, _ := strconv.Atoi(r.FormValue("points_3rd"))
	ptsScorer, _ := strconv.Atoi(r.FormValue("points_scorer"))
	deadlineStr := r.FormValue("tournament_deadline")

	if exact < 0 || outcome < 0 || group < 0 || pts1st < 0 || pts2nd < 0 || pts3rd < 0 || ptsScorer < 0 {
		a.Tmpl.Page(w, "admin", a.adminPageData(r, "", "Points must be non-negative."))
		return
	}

	deadline := a.Cfg.TournamentBetDeadline
	if deadlineStr != "" {
		if t, err := time.Parse("2006-01-02T15:04", deadlineStr); err == nil {
			deadline = t.UTC()
		}
	}

	a.Cfg.Update(exact, outcome, group, pts1st, pts2nd, pts3rd, ptsScorer, deadline)
	_ = a.Cfg.Save()

	a.Tmpl.Page(w, "admin", a.adminPageData(r, "Config saved.", ""))
}
