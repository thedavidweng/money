package providers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thedavidweng/money/internal/config"
)

type bridgeAPI interface {
	CreateBridgeUser(ctx context.Context, externalUserID string) error
	CreateBridgeAuthToken(ctx context.Context, externalUserID string) (BridgeAuthToken, error)
	CreateBridgeConnectSession(ctx context.Context, accessToken string, request BridgeConnectSessionRequest) (BridgeConnectSession, error)
	ListBridgeItems(ctx context.Context, accessToken string) ([]BridgeItem, error)
	ListBridgeAccounts(ctx context.Context, accessToken string, itemID string) ([]BridgeAccount, error)
	ListBridgeTransactions(ctx context.Context, accessToken string, since string) ([]BridgeSyncTransaction, error)
}

type BridgeAuthToken struct {
	AccessToken    string
	UserUUID       string
	ExternalUserID string
}

type BridgeConnectSessionRequest struct {
	UserEmail             string   `json:"user_email"`
	CountryCode           string   `json:"country_code,omitempty"`
	Capabilities          []string `json:"capabilities,omitempty"`
	CallbackURL           string   `json:"callback_url,omitempty"`
	Context               string   `json:"context,omitempty"`
	AccountTypes          string   `json:"account_types,omitempty"`
	AllowAccountSelection *bool    `json:"allow_account_selection,omitempty"`
}

type BridgeConnectSession struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type BridgeItem struct {
	ID           string
	ProviderID   string
	ProviderName string
	Status       string
}

type BridgeAccount struct {
	ID           string
	ItemID       string
	Name         string
	Type         string
	Balance      float64
	Currency     string
	DataAccess   string
	UpdatedAt    string
	ProviderID   string
	ProviderName string
}

type BridgeSyncTransaction struct {
	ID           string
	AccountID    string
	Date         string
	UpdatedAt    string
	Amount       float64
	Description  string
	MerchantName string
	Currency     string
	Deleted      bool
	Future       bool
}

type bridgeProvider struct {
	cfg          config.ProviderConfig
	client       bridgeAPI
	pollInterval time.Duration
}

func newBridgeProvider(cfg config.ProviderConfig) bridgeProvider {
	return bridgeProvider{cfg: cfg, pollInterval: 2 * time.Second}
}

func (p bridgeProvider) Name() string {
	return "bridge"
}

func (p bridgeProvider) ValidateConfig(ctx context.Context) []ConfigDiagnostic {
	for _, field := range []string{"client_id", "client_secret"} {
		if p.cfg.Fields == nil || p.cfg.Fields[field] == "" {
			return []ConfigDiagnostic{{
				Code:     "PROVIDER_CREDENTIALS_MISSING",
				Message:  "bridge credentials are missing.",
				Severity: "warn",
			}}
		}
	}
	if p.cfg.Fields["user_email"] == "" {
		return []ConfigDiagnostic{{
			Code:     "PROVIDER_CONFIG_MISSING",
			Message:  "bridge user_email is missing.",
			Severity: "warn",
		}}
	}
	return nil
}

func (p bridgeProvider) SearchInstitutions(ctx context.Context, query string) ([]Institution, error) {
	return nil, ErrProviderNotImplemented
}

func (p bridgeProvider) CreateLinkSession(ctx context.Context, request LinkRequest) (LinkSession, error) {
	client, err := p.bridgeClient()
	if err != nil {
		return LinkSession{}, err
	}
	externalUserID := p.cfg.Fields["external_user_id"]
	if externalUserID == "" {
		externalUserID = "bridge-" + request.State
		if err := client.CreateBridgeUser(ctx, externalUserID); err != nil {
			return LinkSession{}, err
		}
	}
	token, err := client.CreateBridgeAuthToken(ctx, externalUserID)
	if err != nil {
		return LinkSession{}, err
	}
	existingItems, err := client.ListBridgeItems(ctx, token.AccessToken)
	if err != nil {
		return LinkSession{}, err
	}
	session, err := client.CreateBridgeConnectSession(ctx, token.AccessToken, BridgeConnectSessionRequest{
		UserEmail:    p.cfg.Fields["user_email"],
		CountryCode:  p.cfg.Fields["country_code"],
		Capabilities: providerListField(p.cfg, "capabilities"),
		CallbackURL:  request.RedirectURI,
		Context:      request.State,
		AccountTypes: p.cfg.Fields["account_types"],
	})
	if err != nil {
		return LinkSession{}, err
	}
	return LinkSession{
		Provider:                "bridge",
		URL:                     session.URL,
		State:                   externalUserID,
		ProviderAccessToken:     token.AccessToken,
		ExistingProviderItemIDs: bridgeItemIDs(existingItems),
	}, nil
}

func (p bridgeProvider) ExchangeLinkToken(ctx context.Context, session LinkSession, callback LinkCallback) (LinkedItem, error) {
	client, err := p.bridgeClient()
	if err != nil {
		return LinkedItem{}, err
	}
	seen := map[string]bool{}
	for _, id := range session.ExistingProviderItemIDs {
		seen[id] = true
	}
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	for {
		items, err := client.ListBridgeItems(ctx, session.ProviderAccessToken)
		if err != nil {
			return LinkedItem{}, err
		}
		for _, item := range items {
			if !seen[item.ID] {
				return p.linkedItem(session, item)
			}
		}
		select {
		case <-ctx.Done():
			return LinkedItem{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (p bridgeProvider) Sync(ctx context.Context, item ProviderItem, sink SyncSink) (SyncResult, error) {
	client, err := p.bridgeClient()
	if err != nil {
		return SyncResult{}, err
	}
	externalUserID := string(item.EncryptedAccessToken)
	if item.ExternalUserID != "" {
		externalUserID = item.ExternalUserID
	}
	if externalUserID == "" {
		return SyncResult{}, fmt.Errorf("bridge external user ID is required")
	}
	token, err := client.CreateBridgeAuthToken(ctx, externalUserID)
	if err != nil {
		return SyncResult{}, err
	}
	result := SyncResult{Provider: "bridge", ProviderItemID: item.ID}
	accounts, err := client.ListBridgeAccounts(ctx, token.AccessToken, item.ProviderExternalItemID)
	if err != nil {
		return SyncResult{}, err
	}
	for _, account := range accounts {
		if account.DataAccess == "disabled" {
			continue
		}
		if err := sink.UpsertAccount(ctx, mapBridgeAccount(item.ID, account)); err != nil {
			return SyncResult{}, err
		}
		result.AccountsSeen++
	}
	transactions, err := client.ListBridgeTransactions(ctx, token.AccessToken, item.TransactionCursor)
	if err != nil {
		return SyncResult{}, err
	}
	nextCursor := item.TransactionCursor
	for _, transaction := range transactions {
		if transaction.UpdatedAt > nextCursor {
			nextCursor = transaction.UpdatedAt
		}
		if transaction.Deleted {
			if err := sink.MarkTransactionRemoved(ctx, item.ID, transaction.ID); err != nil {
				return SyncResult{}, err
			}
			result.TransactionsRemoved++
			continue
		}
		if err := sink.UpsertTransaction(ctx, mapBridgeSyncTransaction(item.ID, transaction)); err != nil {
			return SyncResult{}, err
		}
		if item.TransactionCursor == "" {
			result.TransactionsAdded++
		} else {
			result.TransactionsModified++
		}
	}
	result.NextTransactionCursor = nextCursor
	return result, nil
}

func (p bridgeProvider) bridgeClient() (bridgeAPI, error) {
	if p.client != nil {
		return p.client, nil
	}
	return NewBridgeClient(BridgeClientConfig{
		ClientID:     p.cfg.Fields["client_id"],
		ClientSecret: p.cfg.Fields["client_secret"],
		BaseURL:      p.cfg.Fields["base_url"],
	})
}

func (p bridgeProvider) linkedItem(session LinkSession, item BridgeItem) (LinkedItem, error) {
	if item.ID == "" || item.ProviderID == "" || item.ProviderName == "" {
		return LinkedItem{}, fmt.Errorf("bridge item metadata is incomplete")
	}
	institutionID := providerScopedID("bridge", item.ProviderID)
	return LinkedItem{
		Institution: Institution{
			ID:                    institutionID,
			Name:                  item.ProviderName,
			Provider:              "bridge",
			ProviderInstitutionID: item.ProviderID,
		},
		ProviderItem: ProviderItem{
			ID:                     providerScopedID("bridge", item.ID),
			Provider:               "bridge",
			InstitutionID:          institutionID,
			ProviderExternalItemID: item.ID,
			EncryptedAccessToken:   []byte(session.State),
			ExternalUserID:         session.State,
			Status:                 bridgeItemStatus(item.Status),
			Products:               providerListField(p.cfg, "capabilities"),
		},
	}, nil
}

func bridgeItemIDs(items []BridgeItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func bridgeItemStatus(status string) string {
	if status == "0" || strings.EqualFold(status, "active") || strings.EqualFold(status, "ok") {
		return "active"
	}
	return "reconnect_required"
}

func bridgeStringID(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	default:
		return ""
	}
}

func mapBridgeAccount(providerItemID string, account BridgeAccount) FinancialAccount {
	return FinancialAccount{
		ProviderItemID:           providerItemID,
		ProviderAccountID:        account.ID,
		Name:                     account.Name,
		Type:                     bridgeAccountType(account.Type),
		Subtype:                  account.Type,
		CurrentBalanceMinorUnits: minorUnits(account.Balance),
		Currency:                 account.Currency,
		UpdatedAt:                account.UpdatedAt,
	}
}

func mapBridgeSyncTransaction(providerItemID string, transaction BridgeSyncTransaction) Transaction {
	direction := "credit"
	amount := transaction.Amount
	if amount < 0 {
		direction = "debit"
		amount = -amount
	}
	return MapBridgeTransaction(BridgeTransaction{
		ProviderItemID:        providerItemID,
		ProviderTransactionID: transaction.ID,
		ProviderAccountID:     transaction.AccountID,
		Date:                  transaction.Date,
		Amount:                amount,
		Direction:             direction,
		Description:           transaction.Description,
		MerchantName:          transaction.MerchantName,
		Currency:              transaction.Currency,
		Pending:               transaction.Future,
	})
}

func bridgeAccountType(accountType string) string {
	switch accountType {
	case "checking", "savings", "payment":
		return "depository"
	case "card", "credit_card":
		return "credit"
	case "loan":
		return "loan"
	default:
		return "other_asset"
	}
}
