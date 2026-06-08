package handler

import (
	"net/http"
	"sort"
	"time"

	"tournament-games/internal/db"
	"tournament-games/internal/model"
)

type GroupEntry struct {
	Name     string
	Teams    []string
	UserBet  *model.GroupBet
}

type GroupsPageData struct {
	BaseData
	Groups   []GroupEntry
	IsLocked bool
	Deadline time.Time
}

func (a *App) handleGroupsGet(w http.ResponseWriter, r *http.Request) {
	data, err := a.buildGroupsData(r, "")
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	a.Tmpl.Page(w, "group_bets", data)
}

func (a *App) handleGroupsPost(w http.ResponseWriter, r *http.Request) {
	deadline := a.Cfg.TournamentBetDeadline
	if !deadline.IsZero() && time.Now().UTC().After(deadline) {
		data, _ := a.buildGroupsData(r, "")
		data.IsLocked = true
		a.Tmpl.Page(w, "group_bets", data)
		return
	}

	userID := a.currentUserID(r)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}

	for key, vals := range r.Form {
		if len(key) < 6 || key[:6] != "group_" {
			continue
		}
		groupName := key[6:]
		teamName := ""
		if len(vals) > 0 {
			teamName = vals[0]
		}
		if teamName == "" {
			continue
		}
		if err := db.UpsertGroupBet(a.DB, userID, groupName, teamName); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
	}

	data, _ := a.buildGroupsData(r, "Group bets saved!")
	a.Tmpl.Page(w, "group_bets", data)
}

func (a *App) buildGroupsData(r *http.Request, flash string) (GroupsPageData, error) {
	groupTeams, err := db.GetGroupTeams(a.DB)
	if err != nil {
		return GroupsPageData{}, err
	}

	userBets, err := db.GetGroupBetsForUser(a.DB, a.currentUserID(r))
	if err != nil {
		return GroupsPageData{}, err
	}

	// Sort group names A-L.
	names := make([]string, 0, len(groupTeams))
	for g := range groupTeams {
		names = append(names, g)
	}
	sort.Strings(names)

	entries := make([]GroupEntry, 0, len(names))
	for _, name := range names {
		teams := groupTeams[name]
		sort.Strings(teams)
		entries = append(entries, GroupEntry{
			Name:    name,
			Teams:   teams,
			UserBet: userBets[name],
		})
	}

	deadline := a.Cfg.TournamentBetDeadline
	locked := !deadline.IsZero() && time.Now().UTC().After(deadline)

	base := a.baseData(r)
	base.Flash = flash
	return GroupsPageData{
		BaseData: base,
		Groups:   entries,
		IsLocked: locked,
		Deadline: deadline,
	}, nil
}
