package handler

import (
	"net/http"
	"time"

	"tournament-games/internal/db"
	"tournament-games/internal/model"
)

type FixtureWithBet struct {
	Fixture    *model.Fixture
	UserBet    *model.Bet
	Deadline   time.Time
	IsBettable bool
}

type RoundGroup struct {
	Name     string
	Fixtures []FixtureWithBet
}

type FixturesPageData struct {
	BaseData
	Rounds []RoundGroup
}

func (a *App) handleFixtures(w http.ResponseWriter, r *http.Request) {
	userID := a.currentUserID(r)

	fixtures, err := db.GetFixtures(a.DB)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	bets, err := db.GetBetsForUser(a.DB, userID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	roundMap := make(map[string][]FixtureWithBet)
	var roundOrder []string
	seen := make(map[string]bool)

	for _, f := range fixtures {
		fb := FixtureWithBet{
			Fixture:    f,
			UserBet:    bets[f.ID],
			Deadline:   f.BetDeadline(),
			IsBettable: f.IsBettable(),
		}
		if !seen[f.Round] {
			seen[f.Round] = true
			roundOrder = append(roundOrder, f.Round)
		}
		roundMap[f.Round] = append(roundMap[f.Round], fb)
	}

	rounds := make([]RoundGroup, 0, len(roundOrder))
	for _, r := range roundOrder {
		rounds = append(rounds, RoundGroup{Name: r, Fixtures: roundMap[r]})
	}

	a.Tmpl.Page(w, "fixtures", FixturesPageData{
		BaseData: a.baseData(r),
		Rounds:   rounds,
	})
}
