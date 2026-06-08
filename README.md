
  Build: go build -o ./bin/server . → 19MB single binary, no external files needed except betting.db and appsettings.json.

  To run:
  1. Set your API key in appsettings.json → "api_key": "your-key"
  2. Set "admin_username" and "admin_password" in appsettings.json
  3. ./bin/server → starts on port 8080

  What's included:
  - Register/login with sessions (SQLite-backed, no JWT)
  - Match betting with HTMX inline form (closes 5h before kickoff)
  - Scoring: exact(3) + max(0, 5−|goalDiff|) or outcome(1) + bonus
  - Group winner bets (A–L) and tournament bets (1st/2nd/3rd/top scorer), both with configurable deadline
  - Leaderboard (auto-refreshes every 60s via HTMX)
  - Cron polling every 20 min for match results (free API tier friendly)
  - Admin panel: re-fetch fixtures, trigger scoring, edit points config
  - All fixtures pre-fetched in 1 API call at startup

  Next step: Add your API key and run /admin/setup to load the fixtures, then share the URL with your friends.