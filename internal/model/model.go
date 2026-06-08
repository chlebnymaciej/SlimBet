package model

import "time"

type User struct {
	ID           int64
	Username     string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    time.Time
}

type Fixture struct {
	ID            int64
	APIID         int64
	HomeTeam      string
	AwayTeam      string
	HomeTeamID    int64
	AwayTeamID    int64
	KickoffAt     time.Time
	Round         string
	Group         string
	Venue         string
	Status        string
	GoalsHome     *int
	GoalsAway     *int
	ScoresFetched bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (f *Fixture) IsFinished() bool {
	return f.Status == "FT" || f.Status == "AET" || f.Status == "PEN"
}

func (f *Fixture) BetDeadline() time.Time {
	return f.KickoffAt.Add(-5 * time.Hour)
}

func (f *Fixture) IsBettable() bool {
	return time.Now().UTC().Before(f.BetDeadline())
}

type Bet struct {
	ID        int64
	UserID    int64
	FixtureID int64
	HomeScore int
	AwayScore int
	Points    *int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TournamentBet struct {
	ID          int64
	UserID      int64
	FirstPlace  string
	SecondPlace string
	ThirdPlace  string
	TopScorer   string
	Points      *int
	Locked      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type GroupBet struct {
	ID        int64
	UserID    int64
	GroupName string
	TeamName  string
	Points    *int
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LeaderboardEntry struct {
	Rank             int
	Username         string
	TotalPoints      int
	MatchPoints      int
	TournamentPoints int
	GroupPoints      int
}
