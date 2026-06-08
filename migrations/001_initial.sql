PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version    INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT    NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT    NOT NULL,
    is_admin      INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sessions (
    token  TEXT  PRIMARY KEY,
    data   BLOB  NOT NULL,
    expiry REAL  NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expiry);

CREATE TABLE IF NOT EXISTS fixtures (
    id             INTEGER PRIMARY KEY,
    api_id         INTEGER NOT NULL UNIQUE,
    home_team      TEXT    NOT NULL,
    away_team      TEXT    NOT NULL,
    home_team_id   INTEGER NOT NULL,
    away_team_id   INTEGER NOT NULL,
    kickoff_at     TEXT    NOT NULL,
    round          TEXT    NOT NULL,
    group_name     TEXT    NOT NULL DEFAULT '',
    venue          TEXT    NOT NULL DEFAULT '',
    status         TEXT    NOT NULL DEFAULT 'NS',
    goals_home     INTEGER,
    goals_away     INTEGER,
    scores_fetched INTEGER NOT NULL DEFAULT 0,
    created_at     TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at     TEXT    NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS fixtures_kickoff_idx ON fixtures(kickoff_at);
CREATE INDEX IF NOT EXISTS fixtures_status_idx  ON fixtures(status);
CREATE INDEX IF NOT EXISTS fixtures_round_idx   ON fixtures(round);

CREATE TABLE IF NOT EXISTS bets (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fixture_id INTEGER NOT NULL REFERENCES fixtures(id) ON DELETE CASCADE,
    home_score INTEGER NOT NULL,
    away_score INTEGER NOT NULL,
    points     INTEGER,
    created_at TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, fixture_id)
);
CREATE INDEX IF NOT EXISTS bets_user_idx    ON bets(user_id);
CREATE INDEX IF NOT EXISTS bets_fixture_idx ON bets(fixture_id);

CREATE TABLE IF NOT EXISTS tournament_bets (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE UNIQUE,
    first_place  TEXT    NOT NULL DEFAULT '',
    second_place TEXT    NOT NULL DEFAULT '',
    third_place  TEXT    NOT NULL DEFAULT '',
    top_scorer   TEXT    NOT NULL DEFAULT '',
    points       INTEGER,
    locked       INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at   TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS group_bets (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_name TEXT    NOT NULL,
    team_name  TEXT    NOT NULL,
    points     INTEGER,
    created_at TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(user_id, group_name)
);
CREATE INDEX IF NOT EXISTS group_bets_user_idx ON group_bets(user_id);

INSERT OR IGNORE INTO schema_migrations(version) VALUES (1);
