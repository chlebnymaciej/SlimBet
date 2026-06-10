package export

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"sort"

	"tournament-games/internal/config"
	"tournament-games/internal/db"
	"tournament-games/internal/model"
)

func BuildFixturesCSV(database *sql.DB, w io.Writer) error {
	fixtures, err := db.GetFixtures(database)
	if err != nil {
		return err
	}
	users, err := db.GetNonAdminUsers(database)
	if err != nil {
		return err
	}
	allBets, err := db.GetAllBetsMatrix(database)
	if err != nil {
		return err
	}

	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{"Match", "Kickoff", "Round", "Group", "Score"}
	for _, u := range users {
		header = append(header, u.Username)
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, f := range fixtures {
		score := "-"
		if f.GoalsHome != nil && f.GoalsAway != nil {
			score = fmt.Sprintf("%d-%d", *f.GoalsHome, *f.GoalsAway)
		}
		row := []string{
			f.HomeTeam + " vs " + f.AwayTeam,
			f.KickoffAt.Format("2006-01-02 15:04"),
			f.Round,
			f.Group,
			score,
		}
		fixtureBets := allBets[f.ID]
		for _, u := range users {
			row = append(row, formatFixtureBetCell(fixtureBets[u.ID], f))
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return cw.Error()
}

func formatFixtureBetCell(bet *model.Bet, f *model.Fixture) string {
	if bet == nil {
		return ""
	}
	score := fmt.Sprintf("%d-%d", bet.HomeScore, bet.AwayScore)
	if bet.AdvancesPick == "HOME" {
		score += " -> " + f.HomeTeam
	} else if bet.AdvancesPick == "AWAY" {
		score += " -> " + f.AwayTeam
	}
	if bet.Points == nil {
		return score + " (pending)"
	}
	pts := *bet.Points
	if bet.AdvancesPoints != nil {
		pts += *bet.AdvancesPoints
	}
	return fmt.Sprintf("%s (%dpts)", score, pts)
}

func BuildGroupsCSV(database *sql.DB, w io.Writer) error {
	users, err := db.GetNonAdminUsers(database)
	if err != nil {
		return err
	}
	groupBetsMap, err := db.GetAllGroupBetsMatrix(database)
	if err != nil {
		return err
	}
	groupTeams, err := db.GetGroupTeams(database)
	if err != nil {
		return err
	}

	var groupNames []string
	for g := range groupTeams {
		groupNames = append(groupNames, g)
	}
	sort.Strings(groupNames)

	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{"Group"}
	for _, u := range users {
		header = append(header, u.Username)
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	for _, gName := range groupNames {
		row := []string{"Group " + gName}
		for _, u := range users {
			cell := ""
			if bet := groupBetsMap[gName][u.ID]; bet != nil {
				cell = bet.TeamName
			}
			row = append(row, cell)
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return cw.Error()
}

func BuildTournamentCSV(database *sql.DB, cfg *config.Config, w io.Writer) error {
	users, err := db.GetNonAdminUsers(database)
	if err != nil {
		return err
	}
	allTBets, err := db.GetAllTournamentBets(database)
	if err != nil {
		return err
	}
	tResults, err := db.GetTournamentResults(database)
	if err != nil {
		return err
	}

	pts1st, pts2nd, pts3rd, ptsScorer := cfg.GetTournamentPoints()

	tbMap := make(map[int64]*model.TournamentBet, len(allTBets))
	for _, tb := range allTBets {
		tbMap[tb.UserID] = tb
	}

	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := []string{"Category", "Actual"}
	for _, u := range users {
		header = append(header, u.Username)
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	type rowDef struct {
		label  string
		actual string
		pick   func(*model.TournamentBet) string
	}
	defs := []rowDef{
		{
			fmt.Sprintf("Champion (%dpts)", pts1st),
			tResults.Champion,
			func(tb *model.TournamentBet) string {
				if tb == nil {
					return ""
				}
				return tb.FirstPlace
			},
		},
		{
			fmt.Sprintf("2nd Place (%dpts)", pts2nd),
			tResults.RunnerUp,
			func(tb *model.TournamentBet) string {
				if tb == nil {
					return ""
				}
				return tb.SecondPlace
			},
		},
		{
			fmt.Sprintf("3rd Place (%dpts)", pts3rd),
			tResults.ThirdPlace,
			func(tb *model.TournamentBet) string {
				if tb == nil {
					return ""
				}
				return tb.ThirdPlace
			},
		},
		{
			fmt.Sprintf("Top Scorer (%dpts)", ptsScorer),
			tResults.TopScorer,
			func(tb *model.TournamentBet) string {
				if tb == nil {
					return ""
				}
				return tb.TopScorer
			},
		},
	}

	for _, def := range defs {
		row := []string{def.label, def.actual}
		for _, u := range users {
			row = append(row, def.pick(tbMap[u.ID]))
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return cw.Error()
}
