package handler

import (
	"net/http"
	"time"

	"tournament-games/internal/db"
	"tournament-games/internal/model"
)

type TournamentPageData struct {
	BaseData
	Bet      *model.TournamentBet
	IsLocked bool
	Deadline time.Time
}

func (a *App) handleTournamentGet(w http.ResponseWriter, r *http.Request) {
	bet, err := db.GetTournamentBet(a.DB, a.currentUserID(r))
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	deadline := a.Cfg.TournamentBetDeadline
	locked := !deadline.IsZero() && time.Now().UTC().After(deadline)

	a.Tmpl.Page(w, "tournament_bets", TournamentPageData{
		BaseData: a.baseData(r),
		Bet:      bet,
		IsLocked: locked,
		Deadline: deadline,
	})
}

func (a *App) handleTournamentPost(w http.ResponseWriter, r *http.Request) {
	deadline := a.Cfg.TournamentBetDeadline
	if !deadline.IsZero() && time.Now().UTC().After(deadline) {
		a.Tmpl.Page(w, "tournament_bets", TournamentPageData{
			BaseData: a.baseData(r),
			IsLocked: true,
			Deadline: deadline,
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
		Bet:      saved,
		IsLocked: false,
		Deadline: deadline,
	})
}
