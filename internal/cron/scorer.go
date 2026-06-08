package cron

import (
	"database/sql"
	"log"
	"time"

	footballapi "tournament-games/internal/api"
	"tournament-games/internal/config"
	"tournament-games/internal/db"
	"tournament-games/internal/model"
	"tournament-games/internal/scorer"

	robfigcron "github.com/robfig/cron/v3"
)

type Scorer struct {
	db     *sql.DB
	client *footballapi.Client
	cfg    *config.Config
	cron   *robfigcron.Cron
}

func NewScorer(database *sql.DB, client *footballapi.Client, cfg *config.Config) *Scorer {
	return &Scorer{
		db:     database,
		client: client,
		cfg:    cfg,
		cron:   robfigcron.New(),
	}
}

func (s *Scorer) Start() {
	s.cron.AddFunc("1 * * * *", s.scoreFinished)
	s.cron.AddFunc("0 * * * *", s.cleanSessions)
	s.cron.AddFunc("* * * * *", s.lockIfDeadlinePassed)
	s.cron.Start()
}

func (s *Scorer) Stop() {
	s.cron.Stop()
}

func (s *Scorer) ScoreAll() {
	s.scoreFinished()
}

// FetchResultsNow polls the API for ALL fixtures that have kicked off,
// with no time buffer — use from admin panel for immediate result fetching.
func (s *Scorer) FetchResultsNow() {
	candidates, err := db.GetStartedUnscored(s.db)
	if err != nil {
		log.Printf("cron: fetch-now get candidates: %v", err)
		return
	}

	log.Printf("cron: fetch-now polling %d fixture(s) in bulk", len(candidates))
	s.bulkFetchAndProcess(candidates, "fetch-now")

	finished, err := db.GetUnscoredFinished(s.db)
	if err != nil {
		log.Printf("cron: fetch-now get finished: %v", err)
		return
	}
	for _, f := range finished {
		if err := s.awardPoints(f); err != nil {
			log.Printf("cron: fetch-now award points %d: %v", f.ID, err)
		}
	}
}

func (s *Scorer) ScoreOne(fixtureID int64) error {
	return s.scoreFixture(fixtureID)
}

func (s *Scorer) scoreFinished() {
	candidates, err := db.GetUnscored(s.db)
	if err != nil {
		log.Printf("cron: get unscored: %v", err)
		return
	}

	log.Printf("cron: polling %d fixture(s) in bulk", len(candidates))
	s.bulkFetchAndProcess(candidates, "cron")

	finished, err := db.GetUnscoredFinished(s.db)
	if err != nil {
		log.Printf("cron: get unscored finished: %v", err)
		return
	}
	for _, f := range finished {
		if err := s.awardPoints(f); err != nil {
			log.Printf("cron: award points fixture %d: %v", f.ID, err)
		}
	}
}

// bulkFetchAndProcess fetches all candidates in one API call and processes results.
func (s *Scorer) bulkFetchAndProcess(candidates []*model.Fixture, tag string) {
	if len(candidates) == 0 {
		return
	}

	ids := make([]int64, len(candidates))
	for i, f := range candidates {
		ids[i] = f.ID
	}

	items, err := s.client.FetchMatchesByIDs(ids)
	if err != nil {
		log.Printf("cron: %s bulk fetch: %v", tag, err)
		return
	}

	itemMap := make(map[int64]footballapi.MatchItem, len(items))
	for _, it := range items {
		itemMap[it.ID] = it
	}

	for _, f := range candidates {
		it, ok := itemMap[f.ID]
		if !ok {
			continue
		}
		if err := s.processMatchResult(f.ID, &it); err != nil {
			log.Printf("cron: %s process fixture %d: %v", tag, f.ID, err)
		}
	}
}

// processMatchResult applies a fetched match result to the DB and awards points.
// No API call — caller provides the already-fetched MatchItem.
func (s *Scorer) processMatchResult(fixtureID int64, item *footballapi.MatchItem) error {
	if item.Status != "FINISHED" {
		return nil
	}

	goalsHome, goalsAway := 0, 0
	if item.Score.FullTime.Home != nil {
		goalsHome = *item.Score.FullTime.Home
	}
	if item.Score.FullTime.Away != nil {
		goalsAway = *item.Score.FullTime.Away
	}

	if err := db.UpdateFixtureResult(s.db, fixtureID, item.Status, goalsHome, goalsAway,
		item.Score.Duration, item.Score.Winner); err != nil {
		return err
	}

	fixture, err := db.GetFixtureByID(s.db, fixtureID)
	if err != nil || fixture == nil {
		return err
	}

	return s.awardPoints(fixture)
}

// scoreFixture fetches a single match and processes it — used only by ScoreOne (admin).
func (s *Scorer) scoreFixture(fixtureID int64) error {
	item, err := s.client.FetchMatch(fixtureID)
	if err != nil {
		return err
	}
	return s.processMatchResult(fixtureID, item)
}

func (s *Scorer) awardPoints(fixture *model.Fixture) error {
	bets, err := db.GetBetsForFixture(s.db, fixture.ID)
	if err != nil {
		return err
	}

	pointsExact, pointsOutcome, _ := s.cfg.GetPoints()
	for _, bet := range bets {
		pts := scorer.ScoreBet(bet, fixture, pointsExact, pointsOutcome)
		if err := db.UpdateBetPoints(s.db, bet.ID, pts); err != nil {
			log.Printf("cron: update bet %d points: %v", bet.ID, err)
		}

		if fixture.MatchDuration != "" && fixture.MatchDuration != "REGULAR" && bet.AdvancesPick != "" {
			advPts := 0
			if (bet.AdvancesPick == "HOME" && fixture.MatchWinner == "HOME_TEAM") ||
				(bet.AdvancesPick == "AWAY" && fixture.MatchWinner == "AWAY_TEAM") {
				advPts = 5
			}
			if err := db.UpdateBetAdvancesPoints(s.db, bet.ID, advPts); err != nil {
				log.Printf("cron: update advances points bet %d: %v", bet.ID, err)
			}
		}
	}

	return db.MarkScored(s.db, fixture.ID)
}

func (s *Scorer) cleanSessions() {
	store := db.NewSessionStore(s.db)
	if err := store.DeleteExpired(); err != nil {
		log.Printf("cron: clean sessions: %v", err)
	}
}

func (s *Scorer) lockIfDeadlinePassed() {
	deadline := s.cfg.TournamentBetDeadline
	if deadline.IsZero() || time.Now().UTC().Before(deadline) {
		return
	}
	if err := db.LockTournamentBets(s.db); err != nil {
		log.Printf("cron: lock tournament bets: %v", err)
	}
}
