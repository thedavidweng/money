package plaidlogin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/thedavidweng/money/internal/config"
	"github.com/thedavidweng/money/internal/prompt"
)

type LoginOptions struct {
	ConfigPath   string
	Profile      string
	Environment  string
	TeamSelector string
	TeamPrompt   prompt.Selector
	Products     string
	CountryCodes string
	RedirectURI  string
	Force        bool
	CallbackCode string
	RedirectPort int
	CodeVerifier string
	State        string
	HTTPClient   *http.Client
	TokenURL     string
	DashboardURL string
	Now          func() time.Time
}

type LoginResult struct {
	Provider          string `json:"provider"`
	TeamID            string `json:"team_id"`
	Environment       string `json:"environment"`
	KeysWritten       int    `json:"keys_written"`
	CredentialAction  string `json:"credential_action"`
	DashboardAuthPath string `json:"dashboard_auth_path"`
	NextCommand       string `json:"next_command"`
	ConfigPath        string `json:"config_path"`
	EnvPath           string `json:"env_path"`
}

func RunLogin(ctx context.Context, opts LoginOptions) (LoginResult, error) {
	environment := opts.Environment
	if environment == "" {
		environment = "sandbox"
	}
	meta, err := config.ResolveMetadata(config.Options{ConfigPath: opts.ConfigPath, Profile: opts.Profile})
	if err != nil {
		return LoginResult{}, err
	}
	if meta.ReadOnly {
		return LoginResult{}, Error{Code: ErrorReadOnlyViolation, Message: "Plaid Dashboard login would modify local files while read-only mode is enabled"}
	}
	auth, err := ExchangeCode(ctx, TokenClientConfig{
		TokenURL:   opts.TokenURL,
		HTTPClient: opts.HTTPClient,
		Now:        opts.Now,
	}, ExchangeCodeRequest{
		Code:         opts.CallbackCode,
		RedirectURI:  RedirectURI(opts.RedirectPort),
		CodeVerifier: opts.CodeVerifier,
	})
	if err != nil {
		return LoginResult{}, err
	}
	dashboard := NewDashboardClient(DashboardClientConfig{
		BaseURL:    opts.DashboardURL,
		HTTPClient: opts.HTTPClient,
		Token:      auth,
	})
	teams, err := dashboard.ListTeams(ctx)
	if err != nil {
		return LoginResult{}, err
	}
	team, err := selectTeam(teams, opts.TeamSelector, opts.TeamPrompt)
	if err != nil {
		return LoginResult{}, err
	}
	keys, err := dashboard.FetchKeys(ctx, team.TeamID)
	if err != nil {
		return LoginResult{}, err
	}
	secret, err := SecretForEnvironment(keys, environment)
	if err != nil {
		return LoginResult{}, err
	}

	keysWritten := 0
	credentialAction := "written"
	loaded, loadErr := config.Load(config.Options{ConfigPath: meta.ConfigPath, Profile: opts.Profile})
	existingPlaid := loaded.Providers["plaid"].Fields
	if loadErr == nil && existingPlaid["client_id"] != "" && existingPlaid["secret"] != "" && existingPlaid["environment"] == environment && !opts.Force {
		credentialAction = "preserved_existing"
	} else {
		configResult, err := config.ConfigureProvider(meta.ConfigPath, opts.Profile, config.PlaidSpec, map[string]string{
			"client_id": keys.ClientID,
			"secret":    secret,
		}, providerOptions(environment, opts), opts.Force)
		if err != nil {
			return LoginResult{}, Error{Code: "CONFIG_WRITE_FAILED", Message: "Plaid credentials could not be written", Err: err}
		}
		keysWritten = configResult.KeysWritten
	}
	auth.TeamID = team.TeamID
	auth.ClientID = keys.ClientID
	authPath := DashboardAuthPath(meta.ConfigPath)
	if err := WriteAuthFile(authPath, auth); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{
		Provider:          "plaid",
		TeamID:            team.TeamID,
		Environment:       environment,
		KeysWritten:       keysWritten,
		CredentialAction:  credentialAction,
		DashboardAuthPath: authPath,
		NextCommand:       "money link <institution-query>",
		ConfigPath:        meta.ConfigPath,
		EnvPath:           meta.EnvPath,
	}, nil
}

func providerOptions(environment string, opts LoginOptions) map[string]string {
	return map[string]string{
		"environment":   environment,
		"products":      opts.Products,
		"country_codes": opts.CountryCodes,
		"redirect_uri":  opts.RedirectURI,
	}
}

func selectTeam(teams []Team, selector string, teamPrompt prompt.Selector) (Team, error) {
	if selector != "" || len(teams) <= 1 || teamPrompt == nil {
		return SelectTeam(teams, selector)
	}
	choices := make([]prompt.Choice, 0, len(teams))
	for index, team := range teams {
		label := fmt.Sprintf("%d. %s (%s)", index+1, team.Name, team.ClientID)
		choices = append(choices, prompt.Choice{Label: label, Value: team.TeamID})
	}
	selected, err := teamPrompt.Select("Select Plaid team", choices)
	if err != nil {
		return Team{}, err
	}
	return SelectTeam(teams, selected)
}
