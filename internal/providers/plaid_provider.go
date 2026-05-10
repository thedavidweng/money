package providers

import (
	"context"
	"fmt"
	"strings"

	plaid "github.com/plaid/plaid-go/v40/plaid"

	"github.com/thedavidweng/money/internal/config"
)

type plaidClient interface {
	CreateLinkToken(ctx context.Context, request plaid.LinkTokenCreateRequest) (string, error)
	ExchangePublicToken(ctx context.Context, publicToken string) (PlaidPublicTokenExchangeResult, error)
	SearchInstitutions(ctx context.Context, request plaid.InstitutionsSearchRequest) ([]plaid.Institution, error)
	GetAccounts(ctx context.Context, accessToken string) ([]plaid.AccountBase, error)
	SyncTransactions(ctx context.Context, accessToken string, cursor string) (plaid.TransactionsSyncResponse, error)
}

type PlaidPublicTokenExchangeResult struct {
	AccessToken string
	ItemID      string
}

type plaidProvider struct {
	cfg    config.ProviderConfig
	client plaidClient
}

func newPlaidProvider(cfg config.ProviderConfig) plaidProvider {
	return plaidProvider{cfg: cfg}
}

func (p plaidProvider) Name() string {
	return "plaid"
}

func (p plaidProvider) ValidateConfig(ctx context.Context) []ConfigDiagnostic {
	for _, field := range []string{"client_id", "secret"} {
		if p.cfg.Fields == nil || p.cfg.Fields[field] == "" {
			return []ConfigDiagnostic{{
				Code:     "PROVIDER_CREDENTIALS_MISSING",
				Message:  "plaid credentials are missing.",
				Severity: "warn",
			}}
		}
	}
	return nil
}

func (p plaidProvider) SearchInstitutions(ctx context.Context, query string) ([]Institution, error) {
	client, err := p.plaidClient()
	if err != nil {
		return nil, err
	}
	countries, err := plaidCountryCodes(providerListField(p.cfg, "country_codes"))
	if err != nil {
		return nil, err
	}
	products, err := plaidLinkProducts(providerListField(p.cfg, "products"))
	if err != nil {
		return nil, err
	}
	request := plaid.NewInstitutionsSearchRequest(query, countries)
	request.SetProducts(products)
	plaidInstitutions, err := client.SearchInstitutions(ctx, *request)
	if err != nil {
		return nil, err
	}
	institutions := make([]Institution, 0, len(plaidInstitutions))
	for _, institution := range plaidInstitutions {
		institutions = append(institutions, Institution{
			ID:                    providerScopedID("plaid", institution.GetInstitutionId()),
			Name:                  institution.GetName(),
			Provider:              "plaid",
			ProviderInstitutionID: institution.GetInstitutionId(),
		})
	}
	return institutions, nil
}

func (p plaidProvider) CreateLinkSession(ctx context.Context, request LinkRequest) (LinkSession, error) {
	client, err := p.plaidClient()
	if err != nil {
		return LinkSession{}, err
	}
	linkTokenRequest, err := BuildPlaidLinkTokenCreateRequest(PlaidLinkTokenRequestConfig{
		ClientName:    "money",
		Language:      "en",
		ClientUserID:  request.State,
		Products:      providerListField(p.cfg, "products"),
		CountryCodes:  providerListField(p.cfg, "country_codes"),
		RedirectURI:   request.RedirectURI,
		InstitutionID: request.Institution.ProviderInstitutionID,
	})
	if err != nil {
		return LinkSession{}, err
	}
	linkToken, err := client.CreateLinkToken(ctx, linkTokenRequest)
	if err != nil {
		return LinkSession{}, err
	}
	return LinkSession{Provider: "plaid", LinkToken: linkToken, State: request.State}, nil
}

func (p plaidProvider) ExchangeLinkToken(ctx context.Context, session LinkSession, callback LinkCallback) (LinkedItem, error) {
	if callback.State != session.State {
		return LinkedItem{}, fmt.Errorf("Plaid Link callback state does not match session state")
	}
	if callback.PublicToken == "" {
		return LinkedItem{}, fmt.Errorf("Plaid Link callback public token is required")
	}
	if callback.Metadata.Institution.ID == "" || callback.Metadata.Institution.Name == "" {
		return LinkedItem{}, fmt.Errorf("Plaid Link callback institution metadata is required")
	}
	client, err := p.plaidClient()
	if err != nil {
		return LinkedItem{}, err
	}
	exchanged, err := client.ExchangePublicToken(ctx, callback.PublicToken)
	if err != nil {
		return LinkedItem{}, err
	}
	institutionID := providerScopedID("plaid", callback.Metadata.Institution.ID)
	return LinkedItem{
		Institution: Institution{
			ID:                    institutionID,
			Name:                  callback.Metadata.Institution.Name,
			Provider:              "plaid",
			ProviderInstitutionID: callback.Metadata.Institution.ID,
		},
		ProviderItem: ProviderItem{
			ID:                     providerScopedID("plaid", exchanged.ItemID),
			Provider:               "plaid",
			InstitutionID:          institutionID,
			ProviderExternalItemID: exchanged.ItemID,
			EncryptedAccessToken:   []byte(exchanged.AccessToken),
			ExternalUserID:         session.State,
			Status:                 "active",
			Products:               providerListField(p.cfg, "products"),
		},
	}, nil
}

func (p plaidProvider) Sync(ctx context.Context, item ProviderItem, sink SyncSink) (SyncResult, error) {
	client, err := p.plaidClient()
	if err != nil {
		return SyncResult{}, err
	}
	accessToken := string(item.EncryptedAccessToken)
	if accessToken == "" {
		return SyncResult{}, fmt.Errorf("Plaid access token is required")
	}
	result := SyncResult{Provider: "plaid", ProviderItemID: item.ID}
	accounts, err := client.GetAccounts(ctx, accessToken)
	if err != nil {
		return SyncResult{}, err
	}
	for _, account := range accounts {
		if err := sink.UpsertAccount(ctx, mapPlaidSDKAccount(item.ID, account)); err != nil {
			return SyncResult{}, err
		}
		result.AccountsSeen++
	}

	cursor := item.TransactionCursor
	for {
		page, err := client.SyncTransactions(ctx, accessToken, cursor)
		if err != nil {
			return SyncResult{}, err
		}
		for _, transaction := range page.GetAdded() {
			if err := sink.UpsertTransaction(ctx, mapPlaidSDKTransaction(item.ID, transaction)); err != nil {
				return SyncResult{}, err
			}
			result.TransactionsAdded++
		}
		for _, transaction := range page.GetModified() {
			if err := sink.UpsertTransaction(ctx, mapPlaidSDKTransaction(item.ID, transaction)); err != nil {
				return SyncResult{}, err
			}
			result.TransactionsModified++
		}
		for _, removed := range page.GetRemoved() {
			if err := sink.MarkTransactionRemoved(ctx, item.ID, removed.GetTransactionId()); err != nil {
				return SyncResult{}, err
			}
			result.TransactionsRemoved++
		}
		cursor = page.GetNextCursor()
		if !page.GetHasMore() {
			result.NextTransactionCursor = cursor
			return result, nil
		}
	}
}

func (p plaidProvider) plaidClient() (plaidClient, error) {
	if p.client != nil {
		return p.client, nil
	}
	return NewPlaidClient(PlaidClientConfig{
		ClientID:    p.cfg.Fields["client_id"],
		Secret:      p.cfg.Fields["secret"],
		Environment: p.cfg.Fields["environment"],
	})
}

func providerListField(cfg config.ProviderConfig, name string) []string {
	value := cfg.Fields[name]
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		values = append(values, strings.TrimSpace(part))
	}
	return values
}

func providerScopedID(provider string, externalID string) string {
	return provider + ":" + externalID
}

func mapPlaidSDKAccount(providerItemID string, account plaid.AccountBase) FinancialAccount {
	balances := account.GetBalances()
	return MapPlaidAccount(PlaidAccount{
		ProviderItemID:    providerItemID,
		ProviderAccountID: account.GetAccountId(),
		Name:              account.GetName(),
		OfficialName:      account.GetOfficialName(),
		Mask:              account.GetMask(),
		Type:              string(account.GetType()),
		Subtype:           string(account.GetSubtype()),
		CurrentBalance:    balances.GetCurrent(),
		AvailableBalance:  plaidNullableFloat(balances.GetAvailableOk()),
		AvailableCredit:   plaidNullableFloat(balances.GetLimitOk()),
		Currency:          plaidCurrency(balances.GetIsoCurrencyCode(), balances.GetUnofficialCurrencyCode()),
	})
}

func mapPlaidSDKTransaction(providerItemID string, transaction plaid.Transaction) Transaction {
	category, subcategory := plaidCategories(transaction.Category)
	return MapPlaidTransaction(PlaidTransaction{
		ProviderItemID:        providerItemID,
		ProviderTransactionID: transaction.GetTransactionId(),
		ProviderAccountID:     transaction.GetAccountId(),
		Date:                  transaction.GetDate(),
		AuthorizedDate:        plaidNullableString(transaction.GetAuthorizedDateOk()),
		Amount:                transaction.GetAmount(),
		Name:                  transaction.GetName(),
		MerchantName:          transaction.GetMerchantName(),
		Currency:              plaidCurrency(transaction.GetIsoCurrencyCode(), transaction.GetUnofficialCurrencyCode()),
		Pending:               transaction.GetPending(),
		ProviderCategory:      category,
		ProviderSubcategory:   subcategory,
	})
}

func plaidNullableFloat(value *float64, ok bool) *float64 {
	if !ok {
		return nil
	}
	return value
}

func plaidNullableString(value *string, ok bool) *string {
	if !ok {
		return nil
	}
	return value
}

func plaidCurrency(iso string, unofficial string) string {
	if iso != "" {
		return iso
	}
	return unofficial
}

func plaidCategories(categories []string) (*string, *string) {
	if len(categories) == 0 {
		return nil, nil
	}
	category := categories[0]
	var subcategory *string
	if len(categories) > 1 {
		subcategory = &categories[len(categories)-1]
	}
	return &category, subcategory
}
