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
			label := dayMap[dayKey][0].Fixture.KickoffAt.Format("Mon, 02 Jan 2026")
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

// ── Matrix view ───────────────────────────────────────────────────────────────

type MatrixCell struct {
	Bet          *model.Bet
	IsOwn        bool
	CanEdit      bool
	CellClass    string
	FixtureID    int64
	AdvancesTeam string // full team name or ""
}

type MatrixRow struct {
	Fixture    *model.Fixture
	IsBettable bool
	Cells      []MatrixCell
}

type MatrixPageData struct {
	BaseData
	Users []*model.User
	Rows  []MatrixRow
}

func cellClass(bet *model.Bet) string {
	if bet == nil {
		return "bc-none"
	}
	if bet.Points == nil {
		return "bc-pending"
	}
	switch {
	case *bet.Points == 0:
		return "bc-0"
	case *bet.Points <= 4:
		return "bc-low"
	case *bet.Points <= 7:
		return "bc-mid"
	default:
		return "bc-high"
	}
}

func (a *App) handleFixturesMatrix(w http.ResponseWriter, r *http.Request) {
	currentUserID := a.currentUserID(r)

	fixtures, err := db.GetFixtures(a.DB)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	users, err := db.GetNonAdminUsers(a.DB)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	allBets, err := db.GetAllBetsMatrix(a.DB)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	rows := make([]MatrixRow, 0, len(fixtures))
	for _, f := range fixtures {
		bettable := f.IsBettable()
		fixtureBets := allBets[f.ID]

		cells := make([]MatrixCell, len(users))
		for i, u := range users {
			bet := fixtureBets[u.ID]
			isOwn := u.ID == currentUserID

			advancesTeam := ""
			if bet != nil && bet.AdvancesPick != "" {
				if bet.AdvancesPick == "HOME" {
					advancesTeam = f.HomeTeam
				} else {
					advancesTeam = f.AwayTeam
				}
			}

			cells[i] = MatrixCell{
				Bet:          bet,
				IsOwn:        isOwn,
				CanEdit:      isOwn && bettable,
				CellClass:    cellClass(bet),
				FixtureID:    f.ID,
				AdvancesTeam: advancesTeam,
			}
		}

		rows = append(rows, MatrixRow{
			Fixture:    f,
			IsBettable: bettable,
			Cells:      cells,
		})
	}

	a.Tmpl.Page(w, "fixtures_matrix", MatrixPageData{
		BaseData: a.baseData(r),
		Users:    users,
		Rows:     rows,
	})
}
