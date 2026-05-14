package plaidlogin

import (
	"errors"
	"testing"
)

func TestSelectTeamMatchesIndexIDClientIDAndName(t *testing.T) {
	teams := []Team{
		{TeamID: "team_1", ClientID: "client_1", Name: "Acme"},
		{TeamID: "team_2", ClientID: "client_2", Name: "Beta"},
	}
	cases := []struct {
		selector string
		want     string
	}{
		{selector: "2", want: "team_2"},
		{selector: "team_1", want: "team_1"},
		{selector: "client_2", want: "team_2"},
		{selector: "acme", want: "team_1"},
	}
	for _, tc := range cases {
		t.Run(tc.selector, func(t *testing.T) {
			got, err := SelectTeam(teams, tc.selector)
			if err != nil {
				t.Fatalf("SelectTeam: %v", err)
			}
			if got.TeamID != tc.want {
				t.Fatalf("TeamID = %q, want %q", got.TeamID, tc.want)
			}
		})
	}
}

func TestSelectTeamRejectsAmbiguousSelector(t *testing.T) {
	teams := []Team{
		{TeamID: "team_1", ClientID: "client_1", Name: "Same"},
		{TeamID: "team_2", ClientID: "client_2", Name: "same"},
	}
	_, err := SelectTeam(teams, "same")
	var dashErr Error
	if !errors.As(err, &dashErr) || dashErr.Code != ErrorTeamSelectionRequired {
		t.Fatalf("err = %#v", err)
	}
}

func TestSecretForEnvironmentValidatesSupportedEnvironmentAndProvisioning(t *testing.T) {
	keys := Keys{ClientID: "client", Secrets: map[string]string{"sandbox": "sandbox-secret"}}
	secret, err := SecretForEnvironment(keys, "sandbox")
	if err != nil {
		t.Fatalf("SecretForEnvironment sandbox: %v", err)
	}
	if secret != "sandbox-secret" {
		t.Fatalf("secret = %q", secret)
	}
	_, err = SecretForEnvironment(keys, "development")
	if !isPlaidLoginCode(err, ErrorInvalidEnum) {
		t.Fatalf("development err = %#v", err)
	}
	_, err = SecretForEnvironment(keys, "production")
	if !isPlaidLoginCode(err, ErrorPlaidEnvironmentNotProvisioned) {
		t.Fatalf("production err = %#v", err)
	}
}

func isPlaidLoginCode(err error, code string) bool {
	var dashErr Error
	return errors.As(err, &dashErr) && dashErr.Code == code
}
