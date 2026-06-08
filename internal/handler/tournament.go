package handler

import (
	"net/http"
	"sort"
	"time"

	"tournament-games/internal/db"
	"tournament-games/internal/model"
)

type TournamentPageData struct {
	BaseData
	Bet          *model.TournamentBet
	IsLocked     bool
	Deadline     time.Time
	Teams        []string
	Scorers      []string
	Points1st    int
	Points2nd    int
	Points3rd    int
	PointsScorer int
}

func (a *App) loadTeams() []string {
	groupTeams, _ := db.GetGroupTeams(a.DB)
	seen := make(map[string]bool)
	var teams []string
	for _, ts := range groupTeams {
		for _, t := range ts {
			if !seen[t] {
				seen[t] = true
				teams = append(teams, t)
			}
		}
	}
	sort.Strings(teams)
	return teams
}

func (a *App) handleTournamentGet(w http.ResponseWriter, r *http.Request) {
	bet, err := db.GetTournamentBet(a.DB, a.currentUserID(r))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	scorers, _ := db.GetScorerCandidates(a.DB)
	deadline := a.Cfg.TournamentBetDeadline
	locked := !deadline.IsZero() && time.Now().UTC().After(deadline)
	pts1st, pts2nd, pts3rd, ptsScorer := a.Cfg.GetTournamentPoints()

	a.Tmpl.Page(w, "tournament_bets", TournamentPageData{
		BaseData:     a.baseData(r),
		Bet:          bet,
		IsLocked:     locked,
		Deadline:     deadline,
		Teams:        a.loadTeams(),
		Scorers:      scorers,
		Points1st:    pts1st,
		Points2nd:    pts2nd,
		Points3rd:    pts3rd,
		PointsScorer: ptsScorer,
	})
}

func (a *App) handleTournamentPost(w http.ResponseWriter, r *http.Request) {
	deadline := a.Cfg.TournamentBetDeadline
	pts1st, pts2nd, pts3rd, ptsScorer := a.Cfg.GetTournamentPoints()
	scorers, _ := db.GetScorerCandidates(a.DB)
	teams := a.loadTeams()

	if !deadline.IsZero() && time.Now().UTC().After(deadline) {
		a.Tmpl.Page(w, "tournament_bets", TournamentPageData{
			BaseData:     a.baseData(r),
			IsLocked:     true,
			Deadline:     deadline,
			Teams:        teams,
			Scorers:      scorers,
			Points1st:    pts1st,
			Points2nd:    pts2nd,
			Points3rd:    pts3rd,
			PointsScorer: ptsScorer,
			Bet: &model.TournamentBet{
				FirstPlace:  r.FormValue("first_place"),
				SecondPlace: r.FormValue("second_place"),
				ThirdPlace:  r.FormValue("third_place"),
				TopScorer:   r.FormValue("top_scorer"),
			},
		})
		return
	}

	tb := &model.TournamentBet{
		UserID:      a.currentUserID(r),
		FirstPlace:  r.FormValue("first_place"),
		SecondPlace: r.FormValue("second_place"),
		ThirdPlace:  r.FormValue("third_place"),
		TopScorer:   r.FormValue("top_scorer"),
	}

	if err := db.UpsertTournamentBet(a.DB, tb); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	saved, _ := db.GetTournamentBet(a.DB, a.currentUserID(r))
	a.Tmpl.Page(w, "tournament_bets", TournamentPageData{
		BaseData: BaseData{
			Username: a.currentUsername(r),
			IsAdmin:  a.isAdmin(r),
			Flash:    "Tournament bets saved!",
		},
		Bet:          saved,
		IsLocked:     false,
		Deadline:     deadline,
		Teams:        teams,
		Scorers:      scorers,
		Points1st:    pts1st,
		Points2nd:    pts2nd,
		Points3rd:    pts3rd,
		PointsScorer: ptsScorer,
	})
}
