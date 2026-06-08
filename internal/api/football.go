package footballapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func New(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.football-data.org/v4",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ── Response types ────────────────────────────────────────────────────────────

type MatchesResponse struct {
	Matches []MatchItem `json:"matches"`
}

type MatchItem struct {
	ID      int64  `json:"id"`
	UTCDate string `json:"utcDate"` // RFC3339, e.g. "2026-06-11T18:00:00Z"
	Status  string `json:"status"`  // SCHEDULED, IN_PLAY, PAUSED, FINISHED, POSTPONED, CANCELLED
	Stage   string `json:"stage"`   // GROUP_STAGE, ROUND_OF_32, QUARTER_FINALS, etc.
	Group   string `json:"group"`   // GROUP_A … GROUP_L, empty for knockout rounds
	HomeTeam struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"homeTeam"`
	AwayTeam struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"awayTeam"`
	Score struct {
		Winner   string `json:"winner"`   // HOME_TEAM, AWAY_TEAM, DRAW, or empty
		Duration string `json:"duration"` // REGULAR, EXTRA_TIME, PENALTY_SHOOTOUT
		FullTime struct {
			Home *int `json:"home"`
			Away *int `json:"away"`
		} `json:"fullTime"`
	} `json:"score"`
}

// FetchMatches fetches all matches for a competition (e.g. "WC" for World Cup).
func (c *Client) FetchMatches(competitionCode string) ([]MatchItem, error) {
	url := fmt.Sprintf("%s/competitions/%s/matches", c.baseURL, competitionCode)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned %d", resp.StatusCode)
	}

	var result MatchesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return result.Matches, nil
}

// FetchMatch fetches a single match by ID for result polling.
func (c *Client) FetchMatch(id int64) (*MatchItem, error) {
	url := fmt.Sprintf("%s/matches/%d", c.baseURL, id)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth-Token", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned %d", resp.StatusCode)
	}

	var item MatchItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return &item, nil
}
