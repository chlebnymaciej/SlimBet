package handler

import (
	"net/http"
	"strconv"
	"time"

	"tournament-games/internal/db"
	"tournament-games/internal/model"
)

type FixtureWithBet struct {
	Fixture       *model.Fixture
	UserBet       *model.Bet
	Deadline      time.Time
	IsBettable    bool
	TotalBetCount int
}

type DayGroup struct {
	Date     string
	Fixtures []FixtureWithBet
}

type RoundGroup struct {
	Name      string
	DayGroups []DayGroup
}

type FixturesPageData struct {
	BaseData
	Rounds       []RoundGroup
	FixtureCount int
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

	betCounts, _ := db.GetBetCountsPerFixture(a.DB)

	roundMap := make(map[string][]FixtureWithBet)
	var roundOrder []string
	seenRound := make(map[string]bool)

	for _, f := range fixtures {
		fb := FixtureWithBet{
			Fixture:       f,
			UserBet:       bets[f.ID],
			Deadline:      f.BetDeadline(),
			IsBettable:    f.IsBettable(),
			TotalBetCount: betCounts[f.ID],
		}
		if !seenRound[f.Round] {
			seenRound[f.Round] = true
			roundOrder = append(roundOrder, f.Round)
		}
		roundMap[f.Round] = append(roundMap[f.Round], fb)
	}

	rounds := make([]RoundGroup, 0, len(roundOrder))
	for _, roundName := range roundOrder {
		fixturesInRound := roundMap[roundName]

		var dayOrder []string
		dayMap := make(map[string][]FixtureWithBet)
		seenDay := make(map[string]bool)
		for _, fb := range fixturesInRound {
			dayKey := fb.Fixture.KickoffAt.Format("2006-01-02")
			if !seenDay[dayKey] {
				seenDay[dayKey] = true
				dayOrder = append(dayOrder, dayKey)
			}
			dayMap[dayKey] = append(dayMap[dayKey], fb)
		}

		var dayGroups []DayGroup
		for _, dayKey := range dayOrder {
			label := dayMap[dayKey][0].Fixture.KickoffAt.Format("Mon, 02 Jan 2006")
			dayGroups = append(dayGroups, DayGroup{
				Date:     label,
				Fixtures: dayMap[dayKey],
			})
		}

		rounds = append(rounds, RoundGroup{Name: roundName, DayGroups: dayGroups})
	}

	a.Tmpl.Page(w, "fixtures", FixturesPageData{
		BaseData:     a.baseData(r),
		Rounds:       rounds,
		FixtureCount: len(fixtures),
	})
}

func (a *App) handleFixtureBets(w http.ResponseWriter, r *http.Request) {
	fixtureID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	bets, err := db.GetBetsWithUsernamesForFixture(a.DB, fixtureID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	a.Tmpl.Partial(w, "fixture_bets", bets)
}
