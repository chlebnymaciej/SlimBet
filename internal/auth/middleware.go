package auth

import (
	"net/http"

	"github.com/alexedwards/scs/v2"
)

func RequireAuth(sm *scs.SessionManager, basePath string) func(http.Handler) http.Handler {
	loginURL := basePath + "/login"
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if sm.GetInt64(r.Context(), "user_id") == 0 {
				http.Redirect(w, r, loginURL, http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func RequireAdmin(sm *scs.SessionManager, basePath string) func(http.Handler) http.Handler {
	loginURL := basePath + "/login"
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if sm.GetInt64(r.Context(), "user_id") == 0 {
				http.Redirect(w, r, loginURL, http.StatusSeeOther)
				return
			}
			if !sm.GetBool(r.Context(), "is_admin") {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
