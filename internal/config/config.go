package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Config struct {
	APIKey                string    `json:"api_key"`
	DBPath                string    `json:"db_path"`
	Port                  int       `json:"port"`
	PointsExact           int       `json:"points_exact"`
	PointsOutcome         int       `json:"points_outcome"`
	PointsGroupWinner     int       `json:"points_group_winner"`
	TournamentBetDeadline time.Time `json:"tournament_bet_deadline"`
	LeagueID              int       `json:"league_id"`
	Season                int       `json:"season"`
	AdminUsername         string    `json:"admin_username"`
	AdminPassword         string    `json:"admin_password"`
	SessionSecret         string    `json:"session_secret"`

	path string
	mu   sync.RWMutex
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		DBPath:            "./betting.db",
		Port:              8080,
		PointsExact:       3,
		PointsOutcome:     1,
		PointsGroupWinner: 2,
		LeagueID:          1,
		Season:            2026,
		AdminUsername:     "admin",
		SessionSecret:     "change-me-to-a-random-32-byte-string!",
	}
	cfg.path = path

	f, err := os.Open(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("open config: %w", err)
	}
	if err == nil {
		defer f.Close()
		if err := json.NewDecoder(f).Decode(cfg); err != nil {
			return nil, fmt.Errorf("decode config: %w", err)
		}
	}

	if v := os.Getenv("API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("ADMIN_PASSWORD"); v != "" {
		cfg.AdminPassword = v
	}
	if v := os.Getenv("SESSION_SECRET"); v != "" {
		cfg.SessionSecret = v
	}

	return cfg, nil
}

func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, data, 0o644)
}

func (c *Config) Update(pointsExact, pointsOutcome, pointsGroupWinner int, deadline time.Time) {
	c.mu.Lock()
	c.PointsExact = pointsExact
	c.PointsOutcome = pointsOutcome
	c.PointsGroupWinner = pointsGroupWinner
	c.TournamentBetDeadline = deadline
	c.mu.Unlock()
}

func (c *Config) GetPoints() (exact, outcome, groupWinner int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.PointsExact, c.PointsOutcome, c.PointsGroupWinner
}
