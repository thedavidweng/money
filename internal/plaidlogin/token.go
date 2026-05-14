package plaidlogin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TokenClientConfig struct {
	TokenURL   string
	HTTPClient *http.Client
	Now        func() time.Time
}

type ExchangeCodeRequest struct {
	Code         string
	RedirectURI  string
	CodeVerifier string
}

func ExchangeCode(ctx context.Context, cfg TokenClientConfig, request ExchangeCodeRequest) (Auth, error) {
	values := url.Values{}
	values.Set("grant_type", "authorization_code")
	values.Set("code", request.Code)
	values.Set("redirect_uri", request.RedirectURI)
	values.Set("client_id", ClientID)
	values.Set("code_verifier", request.CodeVerifier)
	return tokenRequest(ctx, cfg, values)
}

func RefreshToken(ctx context.Context, cfg TokenClientConfig, refreshToken string) (Auth, error) {
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("refresh_token", refreshToken)
	return tokenRequest(ctx, cfg, values)
}

func tokenRequest(ctx context.Context, cfg TokenClientConfig, values url.Values) (Auth, error) {
	tokenURL := cfg.TokenURL
	if tokenURL == "" {
		tokenURL = TokenURL
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	now := cfg.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return Auth{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return Auth{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return Auth{}, Error{Code: ErrorPlaidDashboardLoginRejected, Message: "Plaid rejected Dashboard login; configure Plaid manually"}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Auth{}, Error{Code: ErrorDashboardContractChanged, Message: fmt.Sprintf("Dashboard OAuth token request returned HTTP %d", resp.StatusCode)}
	}
	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	decoder := json.NewDecoder(resp.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil {
		return Auth{}, Error{Code: ErrorDashboardContractChanged, Message: "Dashboard OAuth token response shape changed", Err: err}
	}
	if body.AccessToken == "" || body.RefreshToken == "" || body.ExpiresIn <= 0 {
		return Auth{}, Error{Code: ErrorDashboardContractChanged, Message: "Dashboard OAuth token response omitted required fields"}
	}
	return Auth{
		AccessToken:  body.AccessToken,
		RefreshToken: body.RefreshToken,
		ExpiresAt:    now().Add(time.Duration(body.ExpiresIn) * time.Second).UTC(),
	}, nil
}
