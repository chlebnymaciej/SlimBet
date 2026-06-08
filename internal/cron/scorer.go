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

func (s *Scorer) ScoreOne(fixtureID int64) error {
	return s.scoreFixture(fixtureID)
}

func (s *Scorer) scoreFinished() {
	candidates, err := db.GetUnscored(s.db)
	if err != nil {
		log.Printf("cron: get unscored: %v", err)
		return
	}

	log.Printf("cron: polling %d fixture(s) for results", len(candidates))
	for _, f := range candidates {
		if err := s.scoreFixture(f.ID); err != nil {
			log.Printf("cron: score fixture %d: %v", f.ID, err)
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Score fixtures already marked finished in DB but not yet awarded points.
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

func (s *Scorer) scoreFixture(fixtureID int64) error {
	item, err := s.client.FetchFixture(fixtureID)
	if err != nil {
		return err
	}

	status := item.Fixture.Status.Short
	if status != "FT" && status != "AET" && status != "PEN" {
		return nil
	}

	goalsHome, goalsAway := 0, 0
	if status == "AET" || status == "PEN" {
		if item.Score.Fulltime.Home != nil {
			goalsHome = *item.Score.Fulltime.Home
		}
		if item.Score.Fulltime.Away != nil {
			goalsAway = *item.Score.Fulltime.Away
		}
	} else {
		if item.Goals.Home != nil {
			goalsHome = *item.Goals.Home
		}
		if item.Goals.Away != nil {
			goalsAway = *item.Goals.Away
		}
	}

	if err := db.UpdateFixtureResult(s.db, fixtureID, status, goalsHome, goalsAway); err != nil {
		return err
	}

	fixture, err := db.GetFixtureByID(s.db, fixtureID)
	if err != nil || fixture == nil {
		return err
	}

	return s.awardPoints(fixture)
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
