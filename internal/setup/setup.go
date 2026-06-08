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

// PrefetchFixtures loads all fixtures from the API into the DB.
// It is idempotent: skips the API call if fixtures are already present.
// Force=true bypasses the check (admin re-fetch).
func PrefetchFixtures(database *sql.DB, client *footballapi.Client, leagueID, season int, force bool) error {
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

	log.Printf("setup: fetching fixtures from API (league=%d season=%d)…", leagueID, season)
	items, err := client.FetchFixtures(leagueID, season)
	if err != nil {
		return fmt.Errorf("fetch fixtures: %w", err)
	}

	log.Printf("setup: received %d fixtures from API", len(items))
	for i, item := range items {
		f := mapFixture(item)
		if err := db.UpsertFixture(database, f); err != nil {
			return fmt.Errorf("upsert fixture %d: %w", f.ID, err)
		}
		// Rate-limit: sleep 200ms between calls during bulk ops (not needed here
		// since this is a single API call, but kept for awareness).
		_ = i
	}

	count, _ := db.FixtureCount(database)
	log.Printf("setup: %d fixtures now in DB", count)
	return nil
}

func mapFixture(item footballapi.FixtureItem) *model.Fixture {
	kickoff := time.Unix(item.Fixture.Timestamp, 0).UTC()
	if item.Fixture.Timestamp == 0 && item.Fixture.Date != "" {
		if t, err := time.Parse(time.RFC3339, item.Fixture.Date); err == nil {
			kickoff = t.UTC()
		}
	}

	group := item.League.Group
	// Normalize "Group A" → "A"
	if strings.HasPrefix(group, "Group ") {
		group = strings.TrimPrefix(group, "Group ")
	}

	f := &model.Fixture{
		ID:         item.Fixture.ID,
		APIID:      item.Fixture.ID,
		HomeTeam:   item.Teams.Home.Name,
		AwayTeam:   item.Teams.Away.Name,
		HomeTeamID: item.Teams.Home.ID,
		AwayTeamID: item.Teams.Away.ID,
		KickoffAt:  kickoff,
		Round:      item.League.Round,
		Group:      group,
		Venue:      item.Fixture.Venue.Name,
		Status:     item.Fixture.Status.Short,
		GoalsHome:  item.Goals.Home,
		GoalsAway:  item.Goals.Away,
	}
	if item.Fixture.Status.Short == "FT" ||
		item.Fixture.Status.Short == "AET" ||
		item.Fixture.Status.Short == "PEN" {
		f.ScoresFetched = false // will be scored by cron
	}
	return f
}
