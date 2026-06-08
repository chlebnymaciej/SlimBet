# WC 2026 Betting App

Private score-prediction game for World Cup 2026. Users predict exact scores for each match, earn points, and compete on a leaderboard.

---

## Features

- **Match bets** — predict the exact score for any match before the 5-hour kickoff deadline
- **Knockout advances pick** — optional extra bet on who advances if a knockout match goes to ET/PEN
- **Group winner bets** — one pick per group (A–L), same deadline as tournament bets
- **Tournament bets** — champion, 2nd, 3rd place and top scorer
- **Leaderboard** — ranked by total points, auto-refreshes every 60 s
- **See others' bets** — click 👥 on any match row to reveal all users' predictions
- **Admin panel** — re-fetch fixtures, trigger scoring, auto-score tournament bets, configure points
- **Auto-scoring** — background cron polls the API hourly and scores finished matches automatically

---

## Scoring

| Bet type | Points |
|----------|--------|
| Exact score | `points_exact` (default 5) + goal bonus |
| Correct outcome (home/draw/away) | `points_outcome` (default 5) + goal bonus |
| Goal accuracy bonus | `max(0, points_exact - abs(predictedTotal - actualTotal))` |
| Group winner | `points_group_winner` (default 10) |
| Knockout advances pick (ET/PEN) | 5 pts if correct |
| Tournament 1st place | `points_1st` (default 80) |
| Tournament 2nd place | `points_2nd` (default 50) |
| Tournament 3rd place | `points_3rd` (default 30) |
| Top scorer | `points_top_scorer` (default 50) |

All point values are configurable from the Admin panel.

---

## Setup

Before running the app you need two things: a free API key and a few values filled in `appsettings.json`.

### 1. Get a football-data.org API key

1. Go to **[https://www.football-data.org/client/register](https://www.football-data.org/client/register)** and create a free account.
2. After confirming your email, log in and open **My Account → API Keys**.
3. Copy the token shown there — it looks like `abc123def456...`.
4. Paste it as `"api_key"` in `appsettings.json`.

> **Free tier limits:** 10 requests/minute, 100 requests/day. The app uses 1 request to load all fixtures on first start, then 1 bulk request per hourly scoring run — well within the free limit for a private group.

### 2. Configure `appsettings.json`

Open `appsettings.json` and fill in the following fields before the first run:

| Field | What to set |
|-------|-------------|
| `api_key` | Your football-data.org API token |
| `admin_password` | Password for the built-in `admin` account |
| `session_secret` | Any random string ≥ 32 characters (e.g. generate with `openssl rand -hex 32`) |

Everything else has working defaults. The full field reference is in the [Configuration](#configuration----appsettingsjson) section below.

### 3. (Docker only) Set the DB path

When running with Docker Compose, the database must be stored in the mounted volume:

```json
"db_path": "/data/betting.db"
```

### 4. First-run checklist

After starting the app and logging in as admin:

- [ ] **Admin → Re-fetch all fixtures from API** — loads all 104 WC 2026 matches (if not auto-loaded on start)
- [ ] **Admin → Refresh top scorers from API** — populates the Top Scorer dropdown for users
- [ ] Share the URL with your group and have everyone register

---

## Quick Start — Local

**Prerequisites:** Go 1.22+, internet access (football-data.org API)

```bash
# 1. Clone and enter the repo
cd tournament-games

# 2. Edit appsettings.json — fill in api_key, admin_password, session_secret

# 3. Build
go build -o ./bin/server .

# 4. Run
./bin/server
# → http://localhost:8080/
```

On first start the app will:
1. Create `betting.db` (SQLite; all data lives here)
2. Run SQL migrations automatically
3. Create the admin user from `admin_username` / `admin_password` in config
4. Fetch all WC 2026 fixtures from the API (single API call)

If fixtures are not loaded: log in as admin → Admin panel → **Re-fetch all fixtures from API**.

---

## Quick Start — Docker

```bash
# 1. Edit appsettings.json:
#    - Set api_key, admin_password, session_secret
#    - Set "db_path": "/data/betting.db"

# 2. Start
docker compose up -d

# App is at http://localhost:8080/
```

`appsettings.json` is bind-mounted read-only from the host so you can edit config without rebuilding. The database lives in a named Docker volume (`betting_data`).

### Subpath deployment

Set `"base_path": "/betting"` (no trailing slash) in `appsettings.json`. The app serves all URLs under `/betting/...` and handles prefix stripping internally — useful behind an nginx reverse proxy.

---

## Configuration — `appsettings.json`

| Field | Default | Description |
|-------|---------|-------------|
| `api_key` | `""` | football-data.org v4 API token |
| `db_path` | `./betting.db` | SQLite file path |
| `port` | `8080` | HTTP listen port |
| `base_path` | `""` | Subpath prefix, e.g. `/betting` |
| `competition_code` | `WC` | football-data.org competition code |
| `admin_username` | `admin` | Admin account created on startup |
| `admin_password` | `""` | Admin password (env `ADMIN_PASSWORD` overrides) |
| `session_secret` | — | Cookie signing key (env `SESSION_SECRET` overrides) |
| `tournament_bet_deadline` | `2026-06-16T00:00:00Z` | Locks tournament + group bets |
| `points_exact` | `5` | Points for exact score prediction |
| `points_outcome` | `5` | Points for correct outcome (not exact score) |
| `points_group_winner` | `10` | Points per correct group winner |
| `points_1st` / `2nd` / `3rd` | `80` / `50` / `30` | Tournament placement points |
| `points_top_scorer` | `50` | Top scorer bet points |

**Environment variable overrides:** `API_KEY`, `ADMIN_PASSWORD`, `SESSION_SECRET`

---

## Admin Workflow

1. **Load fixtures** — Admin → Re-fetch all fixtures from API (re-run after each knockout round to pick up newly scheduled match times)
2. **Load scorer candidates** — Admin → Refresh top scorers from API (populates the top scorer dropdown)
3. **Score matches** — runs automatically every hour; use **⚡ Fetch results NOW** to trigger immediately without waiting
4. **Score tournament bets** — after the tournament ends, fill in champion / 2nd / 3rd / top scorer in Admin → Tournament Scoring, then click **Score all tournament bets**

---

## Code Structure

```
tournament-games/
├── main.go                     # Entry point — wires all packages, starts HTTP + cron
├── appsettings.json            # Runtime config (edited by admin via panel or manually)
├── Dockerfile                  # Multi-stage build: Go 1.25 builder → Alpine runtime
├── docker-compose.yml          # Volume mounts for appsettings.json and DB
├── migrations/
│   ├── 001_initial.sql         # Core tables: users, sessions, fixtures, bets, tournament/group bets
│   ├── 002_knockout.sql        # Adds advances_pick to bets; match_duration/winner to fixtures
│   └── 003_tournament.sql      # Adds top_scorer_candidates and tournament_results tables
├── internal/
│   ├── api/football.go         # football-data.org v4 client: FetchMatches, FetchMatch, FetchScorers
│   ├── auth/                   # bcrypt helpers + RequireAuth / RequireAdmin middleware
│   ├── config/config.go        # Config struct; loads appsettings.json + env overrides; Save()
│   ├── cron/scorer.go          # Hourly cron: polls API for finished matches, awards points
│   ├── db/                     # SQLite connection, migration runner, all SQL query functions
│   ├── handler/                # HTTP handlers, template loading (LoadTemplates), route registration
│   ├── model/model.go          # Plain Go structs: Fixture, Bet, TournamentBet, GroupBet, …
│   ├── scorer/scorer.go        # ScoreBet() — pure function, no DB, fully unit-tested
│   └── setup/setup.go          # PrefetchFixtures — idempotent API-to-DB fixture load
└── web/
    ├── static/style.css        # Custom styles (PicoCSS from CDN + overrides)
    └── templates/              # Go html/template files (embedded in binary at build time)
```

### Package responsibilities

| Package | What it does |
|---------|-------------|
| `internal/api` | HTTP client for football-data.org; typed response structs |
| `internal/db` | Open SQLite, run versioned migrations, all query functions (no ORM) |
| `internal/scorer` | `ScoreBet(bet, fixture, X, Y) int` — pure scoring logic |
| `internal/cron` | Hourly auto-scoring + admin-triggered immediate fetch |
| `internal/handler` | `App` struct wires all handlers; `LoadTemplates` builds the template set with base-path-aware `url` function |
| `internal/setup` | One-time (or admin-forced) fixture pre-fetch from the API |

### Request flow

```
Browser
  → http.StripPrefix          (if base_path configured)
  → scs.SessionManager        (loads/saves session cookie)
  → http.ServeMux             (routes by method + path)
  → RequireAuth/RequireAdmin  (redirect to /login if not authenticated)
  → Handler                   (reads DB, renders html/template or HTMX partial)
```

Templates are embedded in the binary (`//go:embed web`). The `url` template function prepends `base_path` to every internal link automatically.

### Scoring flow

```
Cron (hourly) or admin "Fetch results NOW"
  → db.GetUnscored()               fixtures kicked off >115 min ago, not yet scored
  → api.FetchMatch(id)             poll current status from football-data.org
  → db.UpdateFixtureResult()       persist status, goals, duration, winner
  → scorer.ScoreBet()              compute match points (pure function)
  → db.UpdateBetPoints()           write score + goal-bonus points
  → db.UpdateBetAdvancesPoints()   write ET/PEN advances-pick bonus (5 pts)
  → db.MarkScored()                set scores_fetched = 1
```

### Database tables

| Table | Purpose |
|-------|---------|
| `users` | Accounts (username, bcrypt hash, is_admin flag) |
| `sessions` | SCS session store (token, data, expiry) |
| `fixtures` | All 104 WC 2026 matches cached from football-data.org |
| `bets` | Per-user score predictions + awarded points |
| `tournament_bets` | Champion/2nd/3rd/top-scorer picks per user |
| `group_bets` | Group winner picks — one per group per user |
| `top_scorer_candidates` | Player dropdown, refreshed from API by admin |
| `tournament_results` | Actual final results used to auto-score tournament bets |
| `schema_migrations` | Applied migration version tracking |

---

## Development

```bash
make build    # go build -o ./bin/server .
make run      # build + ./bin/server
make test     # go test ./...
make lint     # go vet ./...
make clean    # rm -rf ./bin/
```

Unit tests cover `internal/scorer` — all scoring combinations. The scorer is a pure function with no DB dependency so tests are fast and deterministic.
