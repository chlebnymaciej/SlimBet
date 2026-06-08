package handler

import (
	"net/http"
	"strings"

	"tournament-games/internal/auth"
	"tournament-games/internal/db"
)

func (a *App) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	a.Tmpl.Page(w, "login", BaseData{})
}

func (a *App) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	user, err := db.GetUserByUsername(a.DB, username)
	if err != nil || user == nil || !auth.CheckPassword(user.PasswordHash, password) {
		a.Tmpl.Page(w, "login", BaseData{Error: "Invalid username or password."})
		return
	}

	a.SM.Put(r.Context(), "user_id", user.ID)
	a.SM.Put(r.Context(), "username", user.Username)
	a.SM.Put(r.Context(), "is_admin", user.IsAdmin)
	http.Redirect(w, r, "/fixtures", http.StatusSeeOther)
}

func (a *App) handleRegisterGet(w http.ResponseWriter, r *http.Request) {
	a.Tmpl.Page(w, "register", BaseData{})
}

func (a *App) handleRegisterPost(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if len(username) < 2 || len(username) > 30 {
		a.Tmpl.Page(w, "register", BaseData{Error: "Username must be 2–30 characters."})
		return
	}
	if len(password) < 6 {
		a.Tmpl.Page(w, "register", BaseData{Error: "Password must be at least 6 characters."})
		return
	}

	existing, err := db.GetUserByUsername(a.DB, username)
	if err != nil {
		a.Tmpl.Page(w, "register", BaseData{Error: "Server error, try again."})
		return
	}
	if existing != nil {
		a.Tmpl.Page(w, "register", BaseData{Error: "Username already taken."})
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		a.Tmpl.Page(w, "register", BaseData{Error: "Server error, try again."})
		return
	}

	user, err := db.CreateUser(a.DB, username, hash)
	if err != nil {
		a.Tmpl.Page(w, "register", BaseData{Error: "Could not create account."})
		return
	}

	a.SM.Put(r.Context(), "user_id", user.ID)
	a.SM.Put(r.Context(), "username", user.Username)
	a.SM.Put(r.Context(), "is_admin", false)
	http.Redirect(w, r, "/fixtures", http.StatusSeeOther)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	a.SM.Destroy(r.Context())
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
