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
		baseURL: "https://v3.football.api-sports.io",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ── Response types ────────────────────────────────────────────────────────────

type FixturesResponse struct {
	Response []FixtureItem `json:"response"`
	Errors   interface{}   `json:"errors"`
}

type FixtureItem struct {
	Fixture struct {
		ID        int64  `json:"id"`
		Date      string `json:"date"`
		Timestamp int64  `json:"timestamp"`
		Status    struct {
			Short   string `json:"short"`
			Long    string `json:"long"`
			Elapsed *int   `json:"elapsed"`
		} `json:"status"`
		Venue struct {
			Name string `json:"name"`
			City string `json:"city"`
		} `json:"venue"`
	} `json:"fixture"`
	League struct {
		ID    int    `json:"id"`
		Round string `json:"round"`
		Group string `json:"group"`
	} `json:"league"`
	Teams struct {
		Home struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"home"`
		Away struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"away"`
	} `json:"teams"`
	Goals struct {
		Home *int `json:"home"`
		Away *int `json:"away"`
	} `json:"goals"`
	Score struct {
		Fulltime struct {
			Home *int `json:"home"`
			Away *int `json:"away"`
		} `json:"fulltime"`
	} `json:"score"`
}

// FetchFixtures fetches all fixtures for a given league and season.
func (c *Client) FetchFixtures(leagueID, season int) ([]FixtureItem, error) {
	url := fmt.Sprintf("%s/fixtures?league=%d&season=%d", c.baseURL, leagueID, season)
	return c.fetchFixtures(url)
}

// FetchFixture fetches a single fixture by API ID (for result polling).
func (c *Client) FetchFixture(id int64) (*FixtureItem, error) {
	url := fmt.Sprintf("%s/fixtures?id=%d", c.baseURL, id)
	items, err := c.fetchFixtures(url)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("fixture %d not found", id)
	}
	return &items[0], nil
}

func (c *Client) fetchFixtures(url string) ([]FixtureItem, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api returned %d", resp.StatusCode)
	}

	var result FixturesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	return result.Response, nil
}
