package db

import (
	"database/sql"
	"fmt"
	"time"

	footballapi "tournament-games/internal/api"
	"tournament-games/internal/model"
)

// ── Users ────────────────────────────────────────────────────────────────────

func CreateUser(db *sql.DB, username, passwordHash string) (*model.User, error) {
	res, err := db.Exec(
		"INSERT INTO users (username, password_hash) VALUES (?, ?)",
		username, passwordHash,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &model.User{ID: id, Username: username, PasswordHash: passwordHash}, nil
}

func GetUserByUsername(db *sql.DB, username string) (*model.User, error) {
	u := &model.User{}
	err := db.QueryRow(
		"SELECT id, username, password_hash, is_admin FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func GetUserByID(db *sql.DB, id int64) (*model.User, error) {
	u := &model.User{}
	err := db.QueryRow(
		"SELECT id, username, password_hash, is_admin FROM users WHERE id = ?", id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsAdmin)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

// ── Fixtures ─────────────────────────────────────────────────────────────────

func UpsertFixture(db *sql.DB, f *model.Fixture) error {
	_, err := db.Exec(`
		INSERT INTO fixtures
			(id, api_id, home_team, away_team, home_team_id, away_team_id,
			 kickoff_at, round, group_name, venue, status, goals_home, goals_away,
			 scores_fetched, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,datetime('now'))
		ON CONFLICT(id) DO UPDATE SET
			status         = excluded.status,
			goals_home     = excluded.goals_home,
			goals_away     = excluded.goals_away,
			scores_fetched = excluded.scores_fetched,
			updated_at     = datetime('now')`,
		f.ID, f.APIID, f.HomeTeam, f.AwayTeam, f.HomeTeamID, f.AwayTeamID,
		f.KickoffAt.UTC().Format(time.RFC3339), f.Round, f.Group, f.Venue,
		f.Status, f.GoalsHome, f.GoalsAway, boolToInt(f.ScoresFetched),
	)
	return err
}

func GetFixtures(db *sql.DB) ([]*model.Fixture, error) {
	rows, err := db.Query(`
		SELECT id, api_id, home_team, away_team, home_team_id, away_team_id,
		       kickoff_at, round, group_name, venue, status,
		       goals_home, goals_away, scores_fetched, match_duration, match_winner
		FROM fixtures ORDER BY kickoff_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFixtures(rows)
}

func GetFixtureByID(db *sql.DB, id int64) (*model.Fixture, error) {
	row := db.QueryRow(`
		SELECT id, api_id, home_team, away_team, home_team_id, away_team_id,
		       kickoff_at, round, group_name, venue, status,
		       goals_home, goals_away, scores_fetched, match_duration, match_winner
		FROM fixtures WHERE id = ?`, id)
	f, err := scanFixture(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return f, err
}

// GetUnscored returns fixtures that have finished but haven't been scored yet.
func GetUnscored(db *sql.DB) ([]*model.Fixture, error) {
	rows, err := db.Query(`
		SELECT id, api_id, home_team, away_team, home_team_id, away_team_id,
		       kickoff_at, round, group_name, venue, status,
		       goals_home, goals_away, scores_fetched, match_duration, match_winner
		FROM fixtures
		WHERE scores_fetched = 0
		AND status NOT IN ('FINISHED','CANCELLED')
		ORDER BY kickoff_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFixtures(rows)
}

func GetUnscoredFinished(db *sql.DB) ([]*model.Fixture, error) {
	rows, err := db.Query(`
		SELECT id, api_id, home_team, away_team, home_team_id, away_team_id,
		       kickoff_at, round, group_name, venue, status,
		       goals_home, goals_away, scores_fetched, match_duration, match_winner
		FROM fixtures
		WHERE scores_fetched = 0
		  AND status IN ('FINISHED')
		ORDER BY kickoff_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFixtures(rows)
}

// GetStartedUnscored returns fixtures that have kicked off and aren't scored yet,
// with no time buffer — used by the admin "Fetch results NOW" action.
func GetStartedUnscored(db *sql.DB) ([]*model.Fixture, error) {
	rows, err := db.Query(`
		SELECT id, api_id, home_team, away_team, home_team_id, away_team_id,
		       kickoff_at, round, group_name, venue, status,
		       goals_home, goals_away, scores_fetched, match_duration, match_winner
		FROM fixtures
		WHERE scores_fetched = 0
		  AND kickoff_at < datetime('now')
		  AND status NOT IN ('FINISHED','CANCELLED','POSTPONED')
		ORDER BY kickoff_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanFixtures(rows)
}

func FixtureCount(db *sql.DB) (int, error) {
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM fixtures").Scan(&n)
	return n, err
}

func MarkScored(db *sql.DB, fixtureID int64) error {
	_, err := db.Exec(
		"UPDATE fixtures SET scores_fetched=1, updated_at=datetime('now') WHERE id=?",
		fixtureID,
	)
	return err
}

func UpdateFixtureResult(db *sql.DB, fixtureID int64, status string, goalsHome, goalsAway int, duration, winner string) error {
	_, err := db.Exec(`
		UPDATE fixtures SET status=?, goals_home=?, goals_away=?,
		  match_duration=?, match_winner=?, updated_at=datetime('now')
		WHERE id=?`,
		status, goalsHome, goalsAway, duration, winner, fixtureID,
	)
	return err
}

// ── Bets ─────────────────────────────────────────────────────────────────────

func GetBet(db *sql.DB, userID, fixtureID int64) (*model.Bet, error) {
	b := &model.Bet{}
	err := db.QueryRow(`
		SELECT id, user_id, fixture_id, home_score, away_score, points,
		       COALESCE(advances_pick,''), advances_points
		FROM bets WHERE user_id=? AND fixture_id=?`,
		userID, fixtureID,
	).Scan(&b.ID, &b.UserID, &b.FixtureID, &b.HomeScore, &b.AwayScore, &b.Points,
		&b.AdvancesPick, &b.AdvancesPoints)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return b, err
}

func UpsertBet(db *sql.DB, userID, fixtureID int64, homeScore, awayScore int, advancesPick string) error {
	var pick *string
	if advancesPick == "HOME" || advancesPick == "AWAY" {
		pick = &advancesPick
	}
	_, err := db.Exec(`
		INSERT INTO bets (user_id, fixture_id, home_score, away_score, advances_pick)
		VALUES (?,?,?,?,?)
		ON CONFLICT(user_id, fixture_id) DO UPDATE SET
			home_score    = excluded.home_score,
			away_score    = excluded.away_score,
			advances_pick = excluded.advances_pick,
			updated_at    = datetime('now')`,
		userID, fixtureID, homeScore, awayScore, pick,
	)
	return err
}

func GetBetsForFixture(db *sql.DB, fixtureID int64) ([]*model.Bet, error) {
	rows, err := db.Query(`
		SELECT id, user_id, fixture_id, home_score, away_score, points,
		       COALESCE(advances_pick,''), advances_points
		FROM bets WHERE fixture_id=?`, fixtureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bets []*model.Bet
	for rows.Next() {
		b := &model.Bet{}
		if err := rows.Scan(&b.ID, &b.UserID, &b.FixtureID, &b.HomeScore, &b.AwayScore,
			&b.Points, &b.AdvancesPick, &b.AdvancesPoints); err != nil {
			return nil, err
		}
		bets = append(bets, b)
	}
	return bets, rows.Err()
}

func GetBetsForUser(db *sql.DB, userID int64) (map[int64]*model.Bet, error) {
	rows, err := db.Query(`
		SELECT id, user_id, fixture_id, home_score, away_score, points,
		       COALESCE(advances_pick,''), advances_points
		FROM bets WHERE user_id=?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[int64]*model.Bet)
	for rows.Next() {
		b := &model.Bet{}
		if err := rows.Scan(&b.ID, &b.UserID, &b.FixtureID, &b.HomeScore, &b.AwayScore,
			&b.Points, &b.AdvancesPick, &b.AdvancesPoints); err != nil {
			return nil, err
		}
		m[b.FixtureID] = b
	}
	return m, rows.Err()
}

// GetNonAdminUsers returns id+username for all non-admin users, sorted by username.
func GetNonAdminUsers(db *sql.DB) ([]*model.User, error) {
	rows, err := db.Query("SELECT id, username FROM users WHERE is_admin=0 ORDER BY username ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		if err := rows.Scan(&u.ID, &u.Username); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetAllBetsMatrix returns map[fixtureID][userID]*model.Bet for building the matrix view.
func GetAllBetsMatrix(db *sql.DB) (map[int64]map[int64]*model.Bet, error) {
	rows, err := db.Query(`
		SELECT id, user_id, fixture_id, home_score, away_score, points,
		       COALESCE(advances_pick,''), advances_points
		FROM bets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[int64]map[int64]*model.Bet)
	for rows.Next() {
		b := &model.Bet{}
		if err := rows.Scan(&b.ID, &b.UserID, &b.FixtureID, &b.HomeScore, &b.AwayScore,
			&b.Points, &b.AdvancesPick, &b.AdvancesPoints); err != nil {
			return nil, err
		}
		if m[b.FixtureID] == nil {
			m[b.FixtureID] = make(map[int64]*model.Bet)
		}
		m[b.FixtureID][b.UserID] = b
	}
	return m, rows.Err()
}

// GetBetCountsPerFixture returns the number of bets per fixture (all fixtures).
func GetBetCountsPerFixture(db *sql.DB) (map[int64]int, error) {
	rows, err := db.Query("SELECT fixture_id, COUNT(*) FROM bets GROUP BY fixture_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[int64]int)
	for rows.Next() {
		var id int64
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		m[id] = count
	}
	return m, rows.Err()
}

// GetAllGroupBetsMatrix returns map[groupName][userID]*model.GroupBet for the matrix view.
func GetAllGroupBetsMatrix(db *sql.DB) (map[string]map[int64]*model.GroupBet, error) {
	rows, err := db.Query("SELECT id, user_id, group_name, team_name, points FROM group_bets")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]map[int64]*model.GroupBet)
	for rows.Next() {
		gb := &model.GroupBet{}
		if err := rows.Scan(&gb.ID, &gb.UserID, &gb.GroupName, &gb.TeamName, &gb.Points); err != nil {
			return nil, err
		}
		if m[gb.GroupName] == nil {
			m[gb.GroupName] = make(map[int64]*model.GroupBet)
		}
		m[gb.GroupName][gb.UserID] = gb
	}
	return m, rows.Err()
}

// BetWithUser pairs a bet with the bettor's username.
type BetWithUser struct {
	Username       string
	HomeScore      int
	AwayScore      int
	Points         *int
	AdvancesPick   string
	AdvancesPoints *int
}

// GetBetsWithUsernamesForFixture returns all non-admin bets for a fixture with usernames.
func GetBetsWithUsernamesForFixture(db *sql.DB, fixtureID int64) ([]BetWithUser, error) {
	rows, err := db.Query(`
		SELECT u.username, b.home_score, b.away_score, b.points,
		       COALESCE(b.advances_pick,''), b.advances_points
		FROM bets b
		JOIN users u ON u.id = b.user_id
		WHERE b.fixture_id = ? AND u.is_admin = 0
		ORDER BY u.username ASC`, fixtureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bets []BetWithUser
	for rows.Next() {
		var b BetWithUser
		if err := rows.Scan(&b.Username, &b.HomeScore, &b.AwayScore, &b.Points,
			&b.AdvancesPick, &b.AdvancesPoints); err != nil {
			return nil, err
		}
		bets = append(bets, b)
	}
	return bets, rows.Err()
}

func UpdateBetPoints(db *sql.DB, betID int64, points int) error {
	_, err := db.Exec("UPDATE bets SET points=?, updated_at=datetime('now') WHERE id=?", points, betID)
	return err
}

func UpdateBetAdvancesPoints(db *sql.DB, betID int64, points int) error {
	_, err := db.Exec("UPDATE bets SET advances_points=?, updated_at=datetime('now') WHERE id=?", points, betID)
	return err
}

// ── Tournament bets ───────────────────────────────────────────────────────────

func GetTournamentBet(db *sql.DB, userID int64) (*model.TournamentBet, error) {
	tb := &model.TournamentBet{}
	var locked int
	err := db.QueryRow(`
		SELECT id, user_id, first_place, second_place, third_place, top_scorer, points, locked
		FROM tournament_bets WHERE user_id=?`, userID,
	).Scan(&tb.ID, &tb.UserID, &tb.FirstPlace, &tb.SecondPlace, &tb.ThirdPlace, &tb.TopScorer, &tb.Points, &locked)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	tb.Locked = locked == 1
	return tb, err
}

func UpsertTournamentBet(db *sql.DB, tb *model.TournamentBet) error {
	_, err := db.Exec(`
		INSERT INTO tournament_bets (user_id, first_place, second_place, third_place, top_scorer)
		VALUES (?,?,?,?,?)
		ON CONFLICT(user_id) DO UPDATE SET
			first_place  = excluded.first_place,
			second_place = excluded.second_place,
			third_place  = excluded.third_place,
			top_scorer   = excluded.top_scorer,
			updated_at   = datetime('now')`,
		tb.UserID, tb.FirstPlace, tb.SecondPlace, tb.ThirdPlace, tb.TopScorer,
	)
	return err
}

func LockTournamentBets(db *sql.DB) error {
	_, err := db.Exec("UPDATE tournament_bets SET locked=1")
	return err
}

func UpdateTournamentBetPoints(db *sql.DB, userID int64, points int) error {
	_, err := db.Exec(
		"UPDATE tournament_bets SET points=?, updated_at=datetime('now') WHERE user_id=?",
		points, userID,
	)
	return err
}

func GetAllTournamentBets(db *sql.DB) ([]*model.TournamentBet, error) {
	rows, err := db.Query(`
		SELECT id, user_id, first_place, second_place, third_place, top_scorer, points, locked
		FROM tournament_bets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bets []*model.TournamentBet
	for rows.Next() {
		tb := &model.TournamentBet{}
		var locked int
		if err := rows.Scan(&tb.ID, &tb.UserID, &tb.FirstPlace, &tb.SecondPlace, &tb.ThirdPlace, &tb.TopScorer, &tb.Points, &locked); err != nil {
			return nil, err
		}
		tb.Locked = locked == 1
		bets = append(bets, tb)
	}
	return bets, rows.Err()
}

// ── Group bets ────────────────────────────────────────────────────────────────

func GetGroupBetsForUser(db *sql.DB, userID int64) (map[string]*model.GroupBet, error) {
	rows, err := db.Query(`
		SELECT id, user_id, group_name, team_name, points
		FROM group_bets WHERE user_id=?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]*model.GroupBet)
	for rows.Next() {
		gb := &model.GroupBet{}
		if err := rows.Scan(&gb.ID, &gb.UserID, &gb.GroupName, &gb.TeamName, &gb.Points); err != nil {
			return nil, err
		}
		m[gb.GroupName] = gb
	}
	return m, rows.Err()
}

func UpsertGroupBet(db *sql.DB, userID int64, groupName, teamName string) error {
	_, err := db.Exec(`
		INSERT INTO group_bets (user_id, group_name, team_name)
		VALUES (?,?,?)
		ON CONFLICT(user_id, group_name) DO UPDATE SET
			team_name  = excluded.team_name,
			updated_at = datetime('now')`,
		userID, groupName, teamName,
	)
	return err
}

func UpdateGroupBetPoints(db *sql.DB, id int64, points int) error {
	_, err := db.Exec("UPDATE group_bets SET points=?, updated_at=datetime('now') WHERE id=?", points, id)
	return err
}

func GetGroupBetsForGroup(db *sql.DB, groupName string) ([]*model.GroupBet, error) {
	rows, err := db.Query(`
		SELECT id, user_id, group_name, team_name, points
		FROM group_bets WHERE group_name=?`, groupName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bets []*model.GroupBet
	for rows.Next() {
		gb := &model.GroupBet{}
		if err := rows.Scan(&gb.ID, &gb.UserID, &gb.GroupName, &gb.TeamName, &gb.Points); err != nil {
			return nil, err
		}
		bets = append(bets, gb)
	}
	return bets, rows.Err()
}

// ── Groups (derived from fixtures) ───────────────────────────────────────────

// GetGroupTeams returns map[groupName][]teamName derived from fixture data.
func GetGroupTeams(db *sql.DB) (map[string][]string, error) {
	rows, err := db.Query(`
		SELECT DISTINCT group_name, home_team FROM fixtures WHERE group_name != ''
		UNION
		SELECT DISTINCT group_name, away_team FROM fixtures WHERE group_name != ''
		ORDER BY group_name, home_team`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string][]string)
	seen := make(map[string]map[string]bool)
	for rows.Next() {
		var g, t string
		if err := rows.Scan(&g, &t); err != nil {
			return nil, err
		}
		if seen[g] == nil {
			seen[g] = make(map[string]bool)
		}
		if !seen[g][t] {
			seen[g][t] = true
			m[g] = append(m[g], t)
		}
	}
	return m, rows.Err()
}

// ── Leaderboard ───────────────────────────────────────────────────────────────

func GetLeaderboard(db *sql.DB) ([]*model.LeaderboardEntry, error) {
	rows, err := db.Query(`
		SELECT
			u.username,
			COALESCE(SUM(b.points + COALESCE(b.advances_points, 0)), 0) AS match_pts,
			COALESCE(tb.points, 0)      AS tournament_pts,
			COALESCE(SUM(gb.points), 0) AS group_pts
		FROM users u
		LEFT JOIN bets          b  ON b.user_id  = u.id
		LEFT JOIN tournament_bets tb ON tb.user_id = u.id
		LEFT JOIN group_bets    gb ON gb.user_id  = u.id
		WHERE u.is_admin = 0
		GROUP BY u.id
		ORDER BY (COALESCE(SUM(b.points + COALESCE(b.advances_points,0)),0) + COALESCE(tb.points,0) + COALESCE(SUM(gb.points),0)) DESC,
		         u.username ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []*model.LeaderboardEntry
	rank := 1
	for rows.Next() {
		e := &model.LeaderboardEntry{Rank: rank}
		if err := rows.Scan(&e.Username, &e.MatchPoints, &e.TournamentPoints, &e.GroupPoints); err != nil {
			return nil, err
		}
		e.TotalPoints = e.MatchPoints + e.TournamentPoints + e.GroupPoints
		entries = append(entries, e)
		rank++
	}
	return entries, rows.Err()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func scanFixtures(rows *sql.Rows) ([]*model.Fixture, error) {
	var fixtures []*model.Fixture
	for rows.Next() {
		f, err := scanFixtureRow(rows)
		if err != nil {
			return nil, err
		}
		fixtures = append(fixtures, f)
	}
	return fixtures, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanFixture(row *sql.Row) (*model.Fixture, error) {
	f := &model.Fixture{}
	var kickoffStr string
	var scored int
	err := row.Scan(
		&f.ID, &f.APIID, &f.HomeTeam, &f.AwayTeam, &f.HomeTeamID, &f.AwayTeamID,
		&kickoffStr, &f.Round, &f.Group, &f.Venue, &f.Status,
		&f.GoalsHome, &f.GoalsAway, &scored, &f.MatchDuration, &f.MatchWinner,
	)
	if err != nil {
		return nil, err
	}
	f.KickoffAt, _ = time.Parse(time.RFC3339, kickoffStr)
	f.ScoresFetched = scored == 1
	return f, nil
}

func scanFixtureRow(rows *sql.Rows) (*model.Fixture, error) {
	f := &model.Fixture{}
	var kickoffStr string
	var scored int
	err := rows.Scan(
		&f.ID, &f.APIID, &f.HomeTeam, &f.AwayTeam, &f.HomeTeamID, &f.AwayTeamID,
		&kickoffStr, &f.Round, &f.Group, &f.Venue, &f.Status,
		&f.GoalsHome, &f.GoalsAway, &scored, &f.MatchDuration, &f.MatchWinner,
	)
	if err != nil {
		return nil, fmt.Errorf("scan fixture: %w", err)
	}
	f.KickoffAt, _ = time.Parse(time.RFC3339, kickoffStr)
	f.ScoresFetched = scored == 1
	return f, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ── Top scorer candidates ─────────────────────────────────────────────────────

func RefreshScorerCandidates(db *sql.DB, items []footballapi.ScorerItem) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM top_scorer_candidates"); err != nil {
		return err
	}
	for _, item := range items {
		if item.Player.Name == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT OR REPLACE INTO top_scorer_candidates (player_name, team_name, goals, updated_at)
			VALUES (?, ?, ?, datetime('now'))`,
			item.Player.Name, item.Team.Name, item.Goals,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func GetScorerCandidates(db *sql.DB) ([]string, error) {
	rows, err := db.Query(
		"SELECT player_name FROM top_scorer_candidates ORDER BY goals DESC, player_name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

func GetScorerCount(db *sql.DB) (int, error) {
	var n int
	err := db.QueryRow("SELECT COUNT(*) FROM top_scorer_candidates").Scan(&n)
	return n, err
}

// ── Tournament results ────────────────────────────────────────────────────────

type TournamentResults struct {
	Champion, RunnerUp, ThirdPlace, TopScorer string
}

func GetTournamentResults(db *sql.DB) (TournamentResults, error) {
	var r TournamentResults
	err := db.QueryRow(
		"SELECT champion, runner_up, third_place, top_scorer FROM tournament_results WHERE id=1",
	).Scan(&r.Champion, &r.RunnerUp, &r.ThirdPlace, &r.TopScorer)
	if err == sql.ErrNoRows {
		return r, nil
	}
	return r, err
}

// SaveAndScoreTournament saves actual results and awards points to all users.
// Returns the number of bets scored.
func SaveAndScoreTournament(db *sql.DB, r TournamentResults, pts1st, pts2nd, pts3rd, ptsScorer int) (int, error) {
	_, err := db.Exec(`
		UPDATE tournament_results SET champion=?, runner_up=?, third_place=?, top_scorer=? WHERE id=1`,
		r.Champion, r.RunnerUp, r.ThirdPlace, r.TopScorer,
	)
	if err != nil {
		return 0, err
	}

	bets, err := GetAllTournamentBets(db)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, bet := range bets {
		total := 0
		if r.Champion != "" && bet.FirstPlace == r.Champion {
			total += pts1st
		}
		if r.RunnerUp != "" && bet.SecondPlace == r.RunnerUp {
			total += pts2nd
		}
		if r.ThirdPlace != "" && bet.ThirdPlace == r.ThirdPlace {
			total += pts3rd
		}
		if r.TopScorer != "" && bet.TopScorer == r.TopScorer {
			total += ptsScorer
		}
		if err := UpdateTournamentBetPoints(db, bet.UserID, total); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
