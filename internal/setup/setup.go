package setup

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	footballapi "tournament-games/internal/api"
	"tournament-games/internal/db"
	"tournament-games/internal/model"
)

// PrefetchFixtures loads all matches from the API into the DB.
// Idempotent: skips the API call if fixtures are already present unless force=true.
func PrefetchFixtures(database *sql.DB, client *footballapi.Client, competitionCode string, force bool) error {
	if !force {
		count, err := db.FixtureCount(database)
		if err != nil {
			return fmt.Errorf("count fixtures: %w", err)
		}
		if count > 0 {
			log.Printf("setup: %d fixtures already loaded, skipping API fetch", count)
			return nil
		}
	}

	log.Printf("setup: fetching fixtures from API (competition=%s)…", competitionCode)
	items, err := client.FetchMatches(competitionCode)
	if err != nil {
		return fmt.Errorf("fetch matches: %w", err)
	}

	log.Printf("setup: received %d matches from API", len(items))
	for _, item := range items {
		f := mapFixture(item)
		if err := db.UpsertFixture(database, f); err != nil {
			return fmt.Errorf("upsert fixture %d: %w", f.ID, err)
		}
	}

	count, _ := db.FixtureCount(database)
	log.Printf("setup: %d fixtures now in DB", count)
	return nil
}

var stageLabels = map[string]string{
	"GROUP_STAGE":    "Group Stage",
	"ROUND_OF_32":    "Round of 32",
	"LAST_32":        "Round of 32",
	"ROUND_OF_16":    "Round of 16",
	"QUARTER_FINALS": "Quarter Finals",
	"SEMI_FINALS":    "Semi Finals",
	"THIRD_PLACE":    "3rd Place",
	"FINAL":          "Final",
}

func mapFixture(item footballapi.MatchItem) *model.Fixture {
	kickoff, _ := time.Parse(time.RFC3339, item.UTCDate)
	kickoff = kickoff.UTC()

	// Normalize "GROUP_A" → "A", leave knockout groups empty.
	group := strings.TrimPrefix(item.Group, "GROUP_")
	if group == item.Group {
		group = "" // wasn't prefixed, knockout round
	}

	round := stageLabels[item.Stage]
	if round == "" {
		round = item.Stage // fallback: use raw value
	}

	return &model.Fixture{
		ID:         item.ID,
		APIID:      item.ID,
		HomeTeam:   item.HomeTeam.Name,
		AwayTeam:   item.AwayTeam.Name,
		HomeTeamID: item.HomeTeam.ID,
		AwayTeamID: item.AwayTeam.ID,
		KickoffAt:  kickoff,
		Round:      round,
		Group:      group,
		Venue:      "",
		Status:     item.Status,
		GoalsHome:  item.Score.FullTime.Home,
		GoalsAway:  item.Score.FullTime.Away,
	}
}
