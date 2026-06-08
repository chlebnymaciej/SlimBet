package handler

import (
	"net/http"
	"strconv"

	"tournament-games/internal/db"
	"tournament-games/internal/model"
)

type BetFormData struct {
	Fixture     *model.Fixture
	ExistingBet *model.Bet
	IsBettable  bool
	Error       string
}

type FixtureRowData struct {
	Fixture    *model.Fixture
	UserBet    *model.Bet
	IsBettable bool
}

func (a *App) handleBetForm(w http.ResponseWriter, r *http.Request) {
	fixtureID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	fixture, err := db.GetFixtureByID(a.DB, fixtureID)
	if err != nil || fixture == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	bet, _ := db.GetBet(a.DB, a.currentUserID(r), fixtureID)

	a.Tmpl.Partial(w, "bet_form", BetFormData{
		Fixture:     fixture,
		ExistingBet: bet,
		IsBettable:  fixture.IsBettable(),
	})
}

func (a *App) handleBetSubmit(w http.ResponseWriter, r *http.Request) {
	fixtureID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	fixture, err := db.GetFixtureByID(a.DB, fixtureID)
	if err != nil || fixture == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if !fixture.IsBettable() {
		w.WriteHeader(http.StatusConflict)
		a.Tmpl.Partial(w, "bet_form", BetFormData{
			Fixture:    fixture,
			IsBettable: false,
			Error:      "Bet deadline has passed for this match.",
		})
		return
	}

	homeScore, err1 := strconv.Atoi(r.FormValue("home_score"))
	awayScore, err2 := strconv.Atoi(r.FormValue("away_score"))
	if err1 != nil || err2 != nil || homeScore < 0 || awayScore < 0 {
		bet, _ := db.GetBet(a.DB, a.currentUserID(r), fixtureID)
		a.Tmpl.Partial(w, "bet_form", BetFormData{
			Fixture:     fixture,
			ExistingBet: bet,
			IsBettable:  true,
			Error:       "Enter valid non-negative scores.",
		})
		return
	}

	advancesPick := r.FormValue("advances_pick")
	if advancesPick != "HOME" && advancesPick != "AWAY" {
		advancesPick = ""
	}

	userID := a.currentUserID(r)
	if err := db.UpsertBet(a.DB, userID, fixtureID, homeScore, awayScore, advancesPick); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	bet, _ := db.GetBet(a.DB, userID, fixtureID)
	a.Tmpl.Partial(w, "fixture_row", FixtureRowData{
		Fixture:    fixture,
		UserBet:    bet,
		IsBettable: fixture.IsBettable(),
	})
}
