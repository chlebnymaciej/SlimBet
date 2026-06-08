package handler

import (
	"fmt"
	"net/http"
	"sort"
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
	AdvancesTeam string
}

type MatrixRow struct {
	Fixture    *model.Fixture
	IsBettable bool
	Cells      []MatrixCell
}

type GroupMatrixRow struct {
	GroupName string
	Cells     []GroupCell
}

type GroupCell struct {
	TeamName  string
	IsOwn     bool
	CanEdit   bool
	CellClass string
}

type TournamentMatrixRow struct {
	Label  string
	Actual string
	Cells  []TournamentCell
}

type TournamentCell struct {
	Pick      string
	IsOwn     bool
	CanEdit   bool
	CellClass string
}

type MatrixPageData struct {
	BaseData
	Users          []*model.User
	MatchRows      []MatrixRow
	GroupRows      []GroupMatrixRow
	TournamentRows []TournamentMatrixRow
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

func groupCellClass(bet *model.GroupBet) string {
	if bet == nil {
		return "bc-none"
	}
	if bet.Points == nil {
		return "bc-pending"
	}
	if *bet.Points == 0 {
		return "bc-0"
	}
	return "bc-high"
}

func tournamentCellClass(pick, actual string) string {
	if pick == "" {
		return "bc-none"
	}
	if actual == "" {
		return "bc-pending"
	}
	if pick == actual {
		return "bc-high"
	}
	return "bc-0"
}

func (a *App) handleFixturesMatrix(w http.ResponseWriter, r *http.Request) {
	currentUserID := a.currentUserID(r)

	// ── Load all data ────────────────────────────────────────────────────────
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
	groupBetsMap, err := db.GetAllGroupBetsMatrix(a.DB)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	groupTeams, err := db.GetGroupTeams(a.DB)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	allTBets, err := db.GetAllTournamentBets(a.DB)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	tResults, err := db.GetTournamentResults(a.DB)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	isLocked := !a.Cfg.TournamentBetDeadline.IsZero() &&
		time.Now().UTC().After(a.Cfg.TournamentBetDeadline)

	// ── Match rows ───────────────────────────────────────────────────────────
	matchRows := make([]MatrixRow, 0, len(fixtures))
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
		matchRows = append(matchRows, MatrixRow{Fixture: f, IsBettable: bettable, Cells: cells})
	}

	// ── Group rows ───────────────────────────────────────────────────────────
	var groupNames []string
	for g := range groupTeams {
		groupNames = append(groupNames, g)
	}
	sort.Strings(groupNames)

	groupRows := make([]GroupMatrixRow, 0, len(groupNames))
	for _, gName := range groupNames {
		cells := make([]GroupCell, len(users))
		for i, u := range users {
			isOwn := u.ID == currentUserID
			bet := groupBetsMap[gName][u.ID]
			teamName := ""
			if bet != nil {
				teamName = bet.TeamName
			}
			cells[i] = GroupCell{
				TeamName:  teamName,
				IsOwn:     isOwn,
				CanEdit:   isOwn && !isLocked,
				CellClass: groupCellClass(bet),
			}
		}
		groupRows = append(groupRows, GroupMatrixRow{GroupName: gName, Cells: cells})
	}

	// ── Tournament rows ──────────────────────────────────────────────────────
	tbMap := make(map[int64]*model.TournamentBet, len(allTBets))
	for _, tb := range allTBets {
		tbMap[tb.UserID] = tb
	}

	pts1st, pts2nd, pts3rd, ptsScorer := a.Cfg.GetTournamentPoints()

	type tRowDef struct {
		label  string
		actual string
		pick   func(*model.TournamentBet) string
	}
	tDefs := []tRowDef{
		{fmt.Sprintf("🥇 Champion (%d pts)", pts1st), tResults.Champion,
			func(tb *model.TournamentBet) string {
				if tb == nil {
					return ""
				}
				return tb.FirstPlace
			}},
		{fmt.Sprintf("🥈 2nd Place (%d pts)", pts2nd), tResults.RunnerUp,
			func(tb *model.TournamentBet) string {
				if tb == nil {
					return ""
				}
				return tb.SecondPlace
			}},
		{fmt.Sprintf("🥉 3rd Place (%d pts)", pts3rd), tResults.ThirdPlace,
			func(tb *model.TournamentBet) string {
				if tb == nil {
					return ""
				}
				return tb.ThirdPlace
			}},
		{fmt.Sprintf("⚽ Top Scorer (%d pts)", ptsScorer), tResults.TopScorer,
			func(tb *model.TournamentBet) string {
				if tb == nil {
					return ""
				}
				return tb.TopScorer
			}},
	}

	tournamentRows := make([]TournamentMatrixRow, 0, len(tDefs))
	for _, def := range tDefs {
		cells := make([]TournamentCell, len(users))
		for i, u := range users {
			isOwn := u.ID == currentUserID
			pick := def.pick(tbMap[u.ID])
			cells[i] = TournamentCell{
				Pick:      pick,
				IsOwn:     isOwn,
				CanEdit:   isOwn && !isLocked,
				CellClass: tournamentCellClass(pick, def.actual),
			}
		}
		tournamentRows = append(tournamentRows, TournamentMatrixRow{
			Label:  def.label,
			Actual: def.actual,
			Cells:  cells,
		})
	}

	a.Tmpl.Page(w, "fixtures_matrix", MatrixPageData{
		BaseData:       a.baseData(r),
		Users:          users,
		MatchRows:      matchRows,
		GroupRows:      groupRows,
		TournamentRows: tournamentRows,
	})
}
