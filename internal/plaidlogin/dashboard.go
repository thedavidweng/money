package plaidlogin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DashboardAPIURL = "https://api.dashboard.plaid.com"

type Auth struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TeamID       string    `json:"team_id,omitempty"`
	ClientID     string    `json:"client_id,omitempty"`
}

type Team struct {
	TeamID   string
	ClientID string
	Name     string
}

type Keys struct {
	ClientID string
	Secrets  map[string]string
}

type DashboardClientConfig struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      Auth
}

type DashboardClient struct {
	baseURL    string
	httpClient *http.Client
	token      Auth
}

func NewDashboardClient(cfg DashboardClientConfig) DashboardClient {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = DashboardAPIURL
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return DashboardClient{baseURL: baseURL, httpClient: httpClient, token: cfg.Token}
}

func (c DashboardClient) ListTeams(ctx context.Context) ([]Team, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/cli/teams/list", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, classifyDashboard401(resp.Body)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, Error{Code: ErrorDashboardContractChanged, Message: fmt.Sprintf("Dashboard teams request returned HTTP %d", resp.StatusCode)}
	}
	var body struct {
		Teams []struct {
			TeamID   string `json:"team_id"`
			ClientID string `json:"client_id"`
			Name     string `json:"name"`
			Company  string `json:"company"`
		} `json:"teams"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, Error{Code: ErrorDashboardContractChanged, Message: "Dashboard teams response shape changed", Err: err}
	}
	if body.Teams == nil {
		return nil, Error{Code: ErrorDashboardContractChanged, Message: "Dashboard teams response omitted teams"}
	}
	teams := make([]Team, 0, len(body.Teams))
	for _, raw := range body.Teams {
		name := raw.Name
		if name == "" {
			name = raw.Company
		}
		if raw.TeamID == "" || raw.ClientID == "" || name == "" {
			return nil, Error{Code: ErrorDashboardContractChanged, Message: "Dashboard teams response omitted required fields"}
		}
		teams = append(teams, Team{TeamID: raw.TeamID, ClientID: raw.ClientID, Name: name})
	}
	return teams, nil
}

func (c DashboardClient) FetchKeys(ctx context.Context, teamID string) (Keys, error) {
	payload, err := json.Marshal(map[string]string{"team_id": teamID})
	if err != nil {
		return Keys{}, err
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/cli/keys/fetch", payload)
	if err != nil {
		return Keys{}, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Keys{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return Keys{}, classifyDashboard401(resp.Body)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Keys{}, Error{Code: ErrorDashboardContractChanged, Message: fmt.Sprintf("Dashboard keys request returned HTTP %d", resp.StatusCode)}
	}
	var body struct {
		ClientID string              `json:"client_id"`
		Secrets  map[string][]string `json:"secrets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return Keys{}, Error{Code: ErrorDashboardContractChanged, Message: "Dashboard keys response shape changed", Err: err}
	}
	if body.ClientID == "" || body.Secrets == nil {
		return Keys{}, Error{Code: ErrorDashboardContractChanged, Message: "Dashboard keys response omitted required fields"}
	}
	secrets := map[string]string{}
	for env, values := range body.Secrets {
		if len(values) == 0 {
			continue
		}
		secrets[env] = values[0]
	}
	return Keys{ClientID: body.ClientID, Secrets: secrets}, nil
}

func (c DashboardClient) newRequest(ctx context.Context, method string, path string, body []byte) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token.AccessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func classifyDashboard401(body io.Reader) error {
	var payload struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return Error{Code: ErrorDashboardContractChanged, Message: "Dashboard auth error shape changed", Err: err}
	}
	code := payload.Error
	if code == "" {
		code = payload.Code
	}
	switch code {
	case "team_selection_required":
		return Error{Code: ErrorTeamSelectionRequired, Message: "Plaid Dashboard requires team selection"}
	case "api_keys_fetch_required":
		return Error{Code: ErrorAPIKeysFetchRequired, Message: "Plaid Dashboard API keys must be fetched"}
	default:
		return Error{Code: ErrorDashboardContractChanged, Message: "Plaid Dashboard returned an unknown auth error"}
	}
}
