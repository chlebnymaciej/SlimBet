-- Persist match duration/winner for knockout advances-pick scoring
ALTER TABLE fixtures ADD COLUMN match_duration TEXT NOT NULL DEFAULT 'REGULAR';
ALTER TABLE fixtures ADD COLUMN match_winner   TEXT NOT NULL DEFAULT '';

-- Advances pick: which team user thinks advances if match is drawn at 90 min
ALTER TABLE bets ADD COLUMN advances_pick   TEXT;    -- NULL, 'HOME', or 'AWAY'
ALTER TABLE bets ADD COLUMN advances_points INTEGER; -- NULL until scored, then 0 or 5

INSERT OR IGNORE INTO schema_migrations(version) VALUES (2);
