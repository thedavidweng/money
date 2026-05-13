package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	plaid "github.com/plaid/plaid-go/v40/plaid"
)

type PlaidClientConfig struct {
	ClientID    string
	Secret      string
	Environment string
}

type PlaidClient struct {
	APIClient     *plaid.APIClient
	Configuration *plaid.Configuration
}

func NewPlaidClient(cfg PlaidClientConfig) (PlaidClient, error) {
	if cfg.ClientID == "" || cfg.Secret == "" {
		return PlaidClient{}, fmt.Errorf("Plaid client ID and secret are required")
	}

	configuration := plaid.NewConfiguration()
	configuration.AddDefaultHeader("PLAID-CLIENT-ID", cfg.ClientID)
	configuration.AddDefaultHeader("PLAID-SECRET", cfg.Secret)

	switch strings.ToLower(cfg.Environment) {
	case "", "sandbox":
		configuration.UseEnvironment(plaid.Sandbox)
	case "production":
		configuration.UseEnvironment(plaid.Production)
	default:
		return PlaidClient{}, fmt.Errorf("unsupported Plaid environment %q", cfg.Environment)
	}

	return PlaidClient{
		APIClient:     plaid.NewAPIClient(configuration),
		Configuration: configuration,
	}, nil
}

func (c PlaidClient) CreateLinkToken(ctx context.Context, request plaid.LinkTokenCreateRequest) (string, error) {
	response, httpResponse, err := c.APIClient.PlaidApi.LinkTokenCreate(ctx).LinkTokenCreateRequest(request).Execute()
	if err != nil {
		if httpResponse != nil {
			return "", ProviderAPIError{
				Provider:   "plaid",
				StatusCode: httpResponse.StatusCode,
				Code:       httpResponse.Status,
				Message:    err.Error(),
			}
		}
		return "", err
	}
	return response.GetLinkToken(), nil
}

func (c PlaidClient) ExchangePublicToken(ctx context.Context, publicToken string) (PlaidPublicTokenExchangeResult, error) {
	request := plaid.NewItemPublicTokenExchangeRequest(publicToken)
	response, httpResponse, err := c.APIClient.PlaidApi.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(*request).Execute()
	if err != nil {
		if httpResponse != nil {
			return PlaidPublicTokenExchangeResult{}, ProviderAPIError{
				Provider:   "plaid",
				StatusCode: httpResponse.StatusCode,
				Code:       httpResponse.Status,
				Message:    err.Error(),
			}
		}
		return PlaidPublicTokenExchangeResult{}, err
	}
	return PlaidPublicTokenExchangeResult{
		AccessToken: response.GetAccessToken(),
		ItemID:      response.GetItemId(),
	}, nil
}

func (c PlaidClient) SearchInstitutions(ctx context.Context, request plaid.InstitutionsSearchRequest) ([]plaid.Institution, error) {
	response, httpResponse, err := c.APIClient.PlaidApi.InstitutionsSearch(ctx).InstitutionsSearchRequest(request).Execute()
	if err != nil {
		if httpResponse != nil {
			return nil, ProviderAPIError{
				Provider:   "plaid",
				StatusCode: httpResponse.StatusCode,
				Code:       httpResponse.Status,
				Message:    err.Error(),
			}
		}
		return nil, err
	}
	return response.GetInstitutions(), nil
}

func (c PlaidClient) GetAccounts(ctx context.Context, accessToken string) ([]plaid.AccountBase, error) {
	request := plaid.NewAccountsGetRequest(accessToken)
	response, httpResponse, err := c.APIClient.PlaidApi.AccountsGet(ctx).AccountsGetRequest(*request).Execute()
	if err != nil {
		if httpResponse != nil {
			return nil, ProviderAPIError{
				Provider:   "plaid",
				StatusCode: httpResponse.StatusCode,
				Code:       httpResponse.Status,
				Message:    err.Error(),
			}
		}
		return nil, err
	}
	return response.GetAccounts(), nil
}

func (c PlaidClient) SyncTransactions(ctx context.Context, accessToken string, cursor string) (plaid.TransactionsSyncResponse, error) {
	request := plaid.NewTransactionsSyncRequest(accessToken)
	if cursor != "" {
		request.SetCursor(cursor)
	}
	response, httpResponse, err := c.APIClient.PlaidApi.TransactionsSync(ctx).TransactionsSyncRequest(*request).Execute()
	if err != nil {
		if httpResponse != nil {
			return plaid.TransactionsSyncResponse{}, ProviderAPIError{
				Provider:   "plaid",
				StatusCode: httpResponse.StatusCode,
				Code:       httpResponse.Status,
				Message:    err.Error(),
			}
		}
		return plaid.TransactionsSyncResponse{}, err
	}
	return response, nil
}

func (c PlaidClient) GetTransactions(ctx context.Context, accessToken string, startDate string, endDate string) ([]plaid.Transaction, error) {
	request := plaid.NewTransactionsGetRequest(accessToken, startDate, endDate)
	var all []plaid.Transaction
	offset := int32(0)
	for {
		request.Options = &plaid.TransactionsGetRequestOptions{
			Offset: plaid.PtrInt32(offset),
			Count:  plaid.PtrInt32(500),
		}
		response, httpResponse, err := c.APIClient.PlaidApi.TransactionsGet(ctx).TransactionsGetRequest(*request).Execute()
		if err != nil {
			if httpResponse != nil {
				return nil, ProviderAPIError{
					Provider:   "plaid",
					StatusCode: httpResponse.StatusCode,
					Code:       httpResponse.Status,
					Message:    err.Error(),
				}
			}
			return nil, err
		}
		page := response.GetTransactions()
		if len(page) == 0 {
			break
		}
		all = append(all, page...)
		if len(all) >= int(response.GetTotalTransactions()) {
			break
		}
		offset = int32(len(all))
	}
	return all, nil
}

func (c PlaidClient) GetHoldings(ctx context.Context, accessToken string) (plaid.InvestmentsHoldingsGetResponse, error) {
	request := plaid.NewInvestmentsHoldingsGetRequest(accessToken)
	response, httpResponse, err := c.APIClient.PlaidApi.InvestmentsHoldingsGet(ctx).InvestmentsHoldingsGetRequest(*request).Execute()
	if err != nil {
		if httpResponse != nil {
			return plaid.InvestmentsHoldingsGetResponse{}, ProviderAPIError{
				Provider:   "plaid",
				StatusCode: httpResponse.StatusCode,
				Code:       httpResponse.Status,
				Message:    err.Error(),
			}
		}
		return plaid.InvestmentsHoldingsGetResponse{}, err
	}
	return response, nil
}

func (c PlaidClient) GetLiabilities(ctx context.Context, accessToken string) (plaid.LiabilitiesGetResponse, error) {
	request := plaid.NewLiabilitiesGetRequest(accessToken)
	response, httpResponse, err := c.APIClient.PlaidApi.LiabilitiesGet(ctx).LiabilitiesGetRequest(*request).Execute()
	if err != nil {
		if httpResponse != nil {
			return plaid.LiabilitiesGetResponse{}, ProviderAPIError{
				Provider:   "plaid",
				StatusCode: httpResponse.StatusCode,
				Code:       httpResponse.Status,
				Message:    err.Error(),
			}
		}
		return plaid.LiabilitiesGetResponse{}, err
	}
	return response, nil
}

type BridgeClientConfig struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
}

type BridgeClient struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	HTTPClient   *http.Client
}

func NewBridgeClient(cfg BridgeClientConfig) (BridgeClient, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return BridgeClient{}, fmt.Errorf("Bridge client ID and client secret are required")
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.bridgeapi.io/v3"
	}
	return BridgeClient{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		BaseURL:      baseURL,
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c BridgeClient) NewRequest(ctx context.Context, method string, path string, accessToken string, body []byte) (*http.Request, error) {
	baseURL, err := url.Parse(strings.TrimRight(c.BaseURL, "/") + "/")
	if err != nil {
		return nil, err
	}
	relative, err := url.Parse(strings.TrimLeft(path, "/"))
	if err != nil {
		return nil, err
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL.ResolveReference(relative).String(), reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Bridge-Version", "2025-01-15")
	req.Header.Set("Client-Id", c.ClientID)
	req.Header.Set("Client-Secret", c.ClientSecret)
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c BridgeClient) CreateBridgeUser(ctx context.Context, externalUserID string) error {
	body, err := json.Marshal(map[string]string{"external_user_id": externalUserID})
	if err != nil {
		return err
	}
	req, err := c.NewRequest(ctx, http.MethodPost, "/aggregation/users", "", body)
	if err != nil {
		return err
	}
	return c.doBridge(req, nil)
}

func (c BridgeClient) CreateBridgeAuthToken(ctx context.Context, externalUserID string) (BridgeAuthToken, error) {
	body, err := json.Marshal(map[string]string{"external_user_id": externalUserID})
	if err != nil {
		return BridgeAuthToken{}, err
	}
	req, err := c.NewRequest(ctx, http.MethodPost, "/aggregation/authorization/token", "", body)
	if err != nil {
		return BridgeAuthToken{}, err
	}
	var response struct {
		AccessToken string `json:"access_token"`
		User        struct {
			UUID           string `json:"uuid"`
			ExternalUserID string `json:"external_user_id"`
		} `json:"user"`
	}
	if err := c.doBridge(req, &response); err != nil {
		return BridgeAuthToken{}, err
	}
	return BridgeAuthToken{
		AccessToken:    response.AccessToken,
		UserUUID:       response.User.UUID,
		ExternalUserID: response.User.ExternalUserID,
	}, nil
}

func (c BridgeClient) CreateBridgeConnectSession(ctx context.Context, accessToken string, request BridgeConnectSessionRequest) (BridgeConnectSession, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return BridgeConnectSession{}, err
	}
	req, err := c.NewRequest(ctx, http.MethodPost, "/aggregation/connect-sessions", accessToken, body)
	if err != nil {
		return BridgeConnectSession{}, err
	}
	var response BridgeConnectSession
	if err := c.doBridge(req, &response); err != nil {
		return BridgeConnectSession{}, err
	}
	return response, nil
}

func (c BridgeClient) ListBridgeItems(ctx context.Context, accessToken string) ([]BridgeItem, error) {
	req, err := c.NewRequest(ctx, http.MethodGet, "/aggregation/items?limit=50", accessToken, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Resources []struct {
			ID           any    `json:"id"`
			ProviderID   any    `json:"provider_id"`
			ProviderName string `json:"provider_name"`
			StatusCode   any    `json:"status_code"`
			Status       any    `json:"status"`
		} `json:"resources"`
	}
	if err := c.doBridge(req, &response); err != nil {
		return nil, err
	}
	items := make([]BridgeItem, 0, len(response.Resources))
	for _, resource := range response.Resources {
		status := bridgeStringID(resource.StatusCode)
		if status == "" {
			status = bridgeStringID(resource.Status)
		}
		items = append(items, BridgeItem{
			ID:           bridgeStringID(resource.ID),
			ProviderID:   bridgeStringID(resource.ProviderID),
			ProviderName: resource.ProviderName,
			Status:       status,
		})
	}
	return items, nil
}

func (c BridgeClient) ListBridgeAccounts(ctx context.Context, accessToken string, itemID string) ([]BridgeAccount, error) {
	path := "/aggregation/accounts?limit=500"
	if itemID != "" {
		path += "&item_id=" + url.QueryEscape(itemID)
	}
	req, err := c.NewRequest(ctx, http.MethodGet, path, accessToken, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Resources []struct {
			ID           any     `json:"id"`
			ItemID       any     `json:"item_id"`
			Name         string  `json:"name"`
			Type         string  `json:"type"`
			Balance      float64 `json:"balance"`
			Currency     string  `json:"currency_code"`
			DataAccess   string  `json:"data_access"`
			UpdatedAt    string  `json:"updated_at"`
			ProviderID   any     `json:"provider_id"`
			ProviderName string  `json:"provider_name"`
		} `json:"resources"`
	}
	if err := c.doBridge(req, &response); err != nil {
		return nil, err
	}
	accounts := make([]BridgeAccount, 0, len(response.Resources))
	for _, resource := range response.Resources {
		accounts = append(accounts, BridgeAccount{
			ID:           bridgeStringID(resource.ID),
			ItemID:       bridgeStringID(resource.ItemID),
			Name:         resource.Name,
			Type:         resource.Type,
			Balance:      resource.Balance,
			Currency:     resource.Currency,
			DataAccess:   resource.DataAccess,
			UpdatedAt:    resource.UpdatedAt,
			ProviderID:   bridgeStringID(resource.ProviderID),
			ProviderName: resource.ProviderName,
		})
	}
	return accounts, nil
}

func (c BridgeClient) ListBridgeTransactions(ctx context.Context, accessToken string, since string) ([]BridgeSyncTransaction, error) {
	path := "/aggregation/transactions?limit=500"
	if since != "" {
		path += "&since=" + url.QueryEscape(since)
	}
	req, err := c.NewRequest(ctx, http.MethodGet, path, accessToken, nil)
	if err != nil {
		return nil, err
	}
	var response struct {
		Resources []struct {
			ID                  any     `json:"id"`
			AccountID           any     `json:"account_id"`
			Date                string  `json:"date"`
			UpdatedAt           string  `json:"updated_at"`
			Amount              float64 `json:"amount"`
			CleanDescription    string  `json:"clean_description"`
			ProviderDescription string  `json:"provider_description"`
			Currency            string  `json:"currency_code"`
			Deleted             bool    `json:"deleted"`
			Future              bool    `json:"future"`
		} `json:"resources"`
	}
	if err := c.doBridge(req, &response); err != nil {
		return nil, err
	}
	transactions := make([]BridgeSyncTransaction, 0, len(response.Resources))
	for _, resource := range response.Resources {
		description := resource.CleanDescription
		if description == "" {
			description = resource.ProviderDescription
		}
		transactions = append(transactions, BridgeSyncTransaction{
			ID:          bridgeStringID(resource.ID),
			AccountID:   bridgeStringID(resource.AccountID),
			Date:        resource.Date,
			UpdatedAt:   resource.UpdatedAt,
			Amount:      resource.Amount,
			Description: description,
			Currency:    resource.Currency,
			Deleted:     resource.Deleted,
			Future:      resource.Future,
		})
	}
	return transactions, nil
}

func (c BridgeClient) doBridge(req *http.Request, output any) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return ProviderAPIError{
			Provider:   "bridge",
			StatusCode: resp.StatusCode,
			Code:       resp.Status,
			Message:    string(body),
		}
	}
	if output == nil {
		_, err := io.Copy(io.Discard, resp.Body)
		return err
	}
	return json.NewDecoder(resp.Body).Decode(output)
}
