package handler

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"net/http"

	"github.com/alexedwards/scs/v2"

	footballapi "tournament-games/internal/api"
	"tournament-games/internal/auth"
	"tournament-games/internal/config"
)

// ScorerInterface is defined in admin.go; declared here to avoid cycles.
// App holds all handler dependencies.
type App struct {
	DB       *sql.DB
	SM       *scs.SessionManager
	Cfg      *config.Config
	Tmpl     *TemplateSet
	API      *footballapi.Client
	BasePath string
	scorer   interface {
		ScoreAll()
		ScoreOne(int64) error
		FetchResultsNow()
	}
}

func (a *App) RegisterRoutes(mux *http.ServeMux, staticFS embed.FS) {
	mux.Handle("GET /static/", http.FileServerFS(staticFS))

	mux.HandleFunc("GET /", a.handleIndex)
	mux.HandleFunc("GET /login", a.handleLoginGet)
	mux.HandleFunc("POST /login", a.handleLoginPost)
	mux.HandleFunc("GET /register", a.handleRegisterGet)
	mux.HandleFunc("POST /register", a.handleRegisterPost)

	requireAuth := auth.RequireAuth(a.SM, a.BasePath)
	requireAdmin := auth.RequireAdmin(a.SM, a.BasePath)

	mux.Handle("GET /fixtures", requireAuth(http.HandlerFunc(a.handleFixtures)))
	mux.Handle("GET /fixtures/{id}/bets", requireAuth(http.HandlerFunc(a.handleFixtureBets)))
	mux.Handle("GET /fixtures/{id}/bet", requireAuth(http.HandlerFunc(a.handleBetForm)))
	mux.Handle("POST /fixtures/{id}/bet", requireAuth(http.HandlerFunc(a.handleBetSubmit)))
	mux.Handle("GET /tournament", requireAuth(http.HandlerFunc(a.handleTournamentGet)))
	mux.Handle("POST /tournament", requireAuth(http.HandlerFunc(a.handleTournamentPost)))
	mux.Handle("GET /groups", requireAuth(http.HandlerFunc(a.handleGroupsGet)))
	mux.Handle("POST /groups", requireAuth(http.HandlerFunc(a.handleGroupsPost)))
	mux.Handle("GET /leaderboard", requireAuth(http.HandlerFunc(a.handleLeaderboard)))
	mux.Handle("GET /leaderboard/partial", requireAuth(http.HandlerFunc(a.handleLeaderboardPartial)))
	mux.Handle("POST /logout", requireAuth(http.HandlerFunc(a.handleLogout)))

	mux.Handle("GET /admin", requireAdmin(http.HandlerFunc(a.handleAdminGet)))
	mux.Handle("POST /admin/setup", requireAdmin(http.HandlerFunc(a.handleAdminSetup)))
	mux.Handle("POST /admin/score/{id}", requireAdmin(http.HandlerFunc(a.handleAdminScoreOne)))
	mux.Handle("POST /admin/score-all", requireAdmin(http.HandlerFunc(a.handleAdminScoreAll)))
	mux.Handle("POST /admin/fetch-results", requireAdmin(http.HandlerFunc(a.handleAdminFetchResults)))
	mux.Handle("POST /admin/refresh-scorers", requireAdmin(http.HandlerFunc(a.handleAdminRefreshScorers)))
	mux.Handle("POST /admin/score-tournament", requireAdmin(http.HandlerFunc(a.handleAdminScoreTournament)))
	mux.Handle("POST /admin/config", requireAdmin(http.HandlerFunc(a.handleAdminConfig)))
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if a.SM.GetInt64(r.Context(), "user_id") != 0 {
		http.Redirect(w, r, a.BasePath+"/fixtures", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, a.BasePath+"/login", http.StatusSeeOther)
}

// ── session helpers ──────────────────────────────────────────────────────────

func (a *App) currentUserID(r *http.Request) int64 {
	return a.SM.GetInt64(r.Context(), "user_id")
}

func (a *App) currentUsername(r *http.Request) string {
	return a.SM.GetString(r.Context(), "username")
}

func (a *App) isAdmin(r *http.Request) bool {
	return a.SM.GetBool(r.Context(), "is_admin")
}

func (a *App) baseData(r *http.Request) BaseData {
	return BaseData{
		Username: a.currentUsername(r),
		IsAdmin:  a.isAdmin(r),
	}
}

type BaseData struct {
	Username string
	IsAdmin  bool
	Flash    string
	Error    string
}

// ── TemplateSet ──────────────────────────────────────────────────────────────

type TemplateSet struct {
	pages    map[string]*template.Template
	partials map[string]*template.Template
}

func buildFuncMap(basePath string) template.FuncMap {
	return template.FuncMap{
		"deref": func(p *int) int {
			if p == nil {
				return 0
			}
			return *p
		},
		"derefStr": func(p *int) string {
			if p == nil {
				return "—"
			}
			return fmt.Sprint(*p)
		},
		"flag": teamFlag,
		"url": func(path string) string {
			return basePath + path
		},
	}
}

func LoadTemplates(fs embed.FS, basePath string) (*TemplateSet, error) {
	fmap := buildFuncMap(basePath)
	ts := &TemplateSet{
		pages:    make(map[string]*template.Template),
		partials: make(map[string]*template.Template),
	}

	// Pages: base + page-specific + any shared partials needed server-side.
	pageFiles := map[string][]string{
		"login":           {"web/templates/base.html", "web/templates/login.html"},
		"register":        {"web/templates/base.html", "web/templates/register.html"},
		"fixtures":        {"web/templates/base.html", "web/templates/fixtures.html", "web/templates/fixture_row.html"},
		"tournament_bets": {"web/templates/base.html", "web/templates/tournament_bets.html"},
		"group_bets":      {"web/templates/base.html", "web/templates/group_bets.html"},
		"leaderboard":     {"web/templates/base.html", "web/templates/leaderboard.html", "web/templates/leaderboard_rows.html"},
		"admin":           {"web/templates/base.html", "web/templates/admin.html"},
	}
	for name, files := range pageFiles {
		tmpl, err := template.New("").Funcs(fmap).ParseFS(fs, files...)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", name, err)
		}
		ts.pages[name] = tmpl
	}

	// Partials: rendered standalone for HTMX responses.
	partialFiles := map[string][]string{
		"fixture_row":      {"web/templates/fixture_row.html"},
		"bet_form":         {"web/templates/bet_form.html"},
		"leaderboard_rows": {"web/templates/leaderboard_rows.html"},
		"fixture_bets":     {"web/templates/fixture_bets.html"},
	}
	for name, files := range partialFiles {
		tmpl, err := template.New(name).Funcs(fmap).ParseFS(fs, files...)
		if err != nil {
			return nil, fmt.Errorf("parse partial %s: %w", name, err)
		}
		ts.partials[name] = tmpl
	}

	return ts, nil
}

func (ts *TemplateSet) Page(w http.ResponseWriter, name string, data any) {
	tmpl, ok := ts.pages[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (ts *TemplateSet) Partial(w http.ResponseWriter, name string, data any) {
	tmpl, ok := ts.partials[name]
	if !ok {
		http.Error(w, "partial not found: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
