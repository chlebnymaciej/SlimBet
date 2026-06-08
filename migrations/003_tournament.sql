-- Top scorer candidates (refreshed from API by admin)
CREATE TABLE IF NOT EXISTS top_scorer_candidates (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    player_name TEXT    NOT NULL UNIQUE,
    team_name   TEXT    NOT NULL DEFAULT '',
    goals       INTEGER NOT NULL DEFAULT 0,
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

-- Actual tournament results (set by admin to trigger auto-scoring)
CREATE TABLE IF NOT EXISTS tournament_results (
    id          INTEGER PRIMARY KEY,
    champion    TEXT NOT NULL DEFAULT '',
    runner_up   TEXT NOT NULL DEFAULT '',
    third_place TEXT NOT NULL DEFAULT '',
    top_scorer  TEXT NOT NULL DEFAULT ''
);
INSERT OR IGNORE INTO tournament_results(id) VALUES (1);

INSERT OR IGNORE INTO schema_migrations(version) VALUES (3);
