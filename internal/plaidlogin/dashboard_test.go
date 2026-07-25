package plaidlogin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDashboardClientListsTeamsAndFetchesKeys(t *testing.T) {
	var sawTeams, sawKeys bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/cli/teams/list":
			sawTeams = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"teams":[{"team_id":"team_1","client_id":"client_1","company":"Acme","role":"admin"}],"pagination":{"total":1}}`))
		case "/cli/keys/fetch":
			sawKeys = true
			if r.Method != http.MethodPost {
				t.Fatalf("keys method = %s, want POST", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"client_id":"client_1","secrets":{"sandbox":["sandbox-secret"],"production":["prod-secret"]},"request_id":"req_123"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewDashboardClient(&DashboardClientConfig{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		Token:      Auth{AccessToken: "access-token", ExpiresAt: time.Now().Add(time.Hour)},
	})

	teams, err := client.ListTeams(context.Background())
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(teams) != 1 || teams[0].TeamID != "team_1" || teams[0].ClientID != "client_1" || teams[0].Name != "Acme" {
		t.Fatalf("teams = %#v", teams)
	}
	keys, err := client.FetchKeys(context.Background(), "team_1")
	if err != nil {
		t.Fatalf("FetchKeys: %v", err)
	}
	if keys.ClientID != "client_1" || keys.Secrets["sandbox"] != "sandbox-secret" || keys.Secrets["production"] != "prod-secret" {
		t.Fatalf("keys = %#v", keys)
	}
	if !sawTeams || !sawKeys {
		t.Fatalf("sawTeams=%v sawKeys=%v", sawTeams, sawKeys)
	}
}

func TestDashboardClientClassifiesContractDriftAndKnown401Codes(t *testing.T) {
	tests := []struct {
		name string
		body string
		code string
	}{
		{name: "missing teams", body: `{}`, code: ErrorDashboardContractChanged},
		{name: "unknown 401", body: `{"error":"new_dashboard_code"}`, code: ErrorDashboardContractChanged},
		{name: "team selection required", body: `{"error":"team_selection_required"}`, code: ErrorTeamSelectionRequired},
		{name: "api keys fetch required", body: `{"error":"api_keys_fetch_required"}`, code: ErrorAPIKeysFetchRequired},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				status := http.StatusOK
				if strings.Contains(tc.body, "error") {
					status = http.StatusUnauthorized
				}
				w.WriteHeader(status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			client := NewDashboardClient(&DashboardClientConfig{
				BaseURL:    server.URL,
				HTTPClient: server.Client(),
				Token:      Auth{AccessToken: "access-token", ExpiresAt: time.Now().Add(time.Hour)},
			})
			_, err := client.ListTeams(context.Background())
			var dashErr Error
			if !errors.As(err, &dashErr) || dashErr.Code != tc.code {
				t.Fatalf("err = %#v, want code %s", err, tc.code)
			}
		})
	}
}
