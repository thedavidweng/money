package providers

import (
	"context"
	"testing"

	plaid "github.com/plaid/plaid-go/v40/plaid"

	"github.com/thedavidweng/money/internal/config"
)

func TestPlaidProviderCreateLinkSessionCreatesPlaidLinkToken(t *testing.T) {
	client := &fakePlaidLinkTokenClient{linkToken: "link-sandbox-token"}
	provider := plaidProvider{
		cfg: config.ProviderConfig{Fields: map[string]string{
			"client_id":      "client-id",
			"secret":         "secret",
			"environment":    "sandbox",
			"products":       "transactions,liabilities",
			"country_codes":  "US,CA",
			"redirect_uri":   "http://127.0.0.1:4000/callback",
			"institution_id": "ignored",
		}},
		client: client,
	}

	session, err := provider.CreateLinkSession(context.Background(), LinkRequest{
		Institution: Institution{ProviderInstitutionID: "ins_123"},
		RedirectURI: "http://127.0.0.1:4000/callback",
		State:       "state-123",
	})
	if err != nil {
		t.Fatalf("create link session: %v", err)
	}

	if session.Provider != "plaid" || session.LinkToken != "link-sandbox-token" || session.State != "state-123" {
		t.Fatalf("session = %#v", session)
	}
	request := client.request
	if request.ClientName != "money" || request.Language != "en" {
		t.Fatalf("request identity = %q/%q", request.ClientName, request.Language)
	}
	if request.User == nil || request.User.ClientUserId != "state-123" {
		t.Fatalf("client user = %#v", request.User)
	}
	if len(request.Products) != 2 || request.Products[0] != plaid.PRODUCTS_TRANSACTIONS || request.Products[1] != plaid.PRODUCTS_LIABILITIES {
		t.Fatalf("products = %#v", request.Products)
	}
	if len(request.CountryCodes) != 2 || request.CountryCodes[0] != plaid.COUNTRYCODE_US || request.CountryCodes[1] != plaid.COUNTRYCODE_CA {
		t.Fatalf("country codes = %#v", request.CountryCodes)
	}
	if request.InstitutionId == nil || *request.InstitutionId != "ins_123" {
		t.Fatalf("institution id = %#v", request.InstitutionId)
	}
}

func TestPlaidProviderSearchInstitutionsUsesConfiguredProductsAndCountries(t *testing.T) {
	client := &fakePlaidLinkTokenClient{
		institutions: []plaid.Institution{
			*plaid.NewInstitution(
				"ins_123",
				"Bank",
				[]plaid.Products{plaid.PRODUCTS_TRANSACTIONS},
				[]plaid.CountryCode{plaid.COUNTRYCODE_US},
				nil,
				false,
			),
		},
	}
	provider := plaidProvider{
		cfg: config.ProviderConfig{Fields: map[string]string{
			"client_id":     "client-id",
			"secret":        "secret",
			"products":      "transactions",
			"country_codes": "US,CA",
		}},
		client: client,
	}

	institutions, err := provider.SearchInstitutions(context.Background(), "bank")
	if err != nil {
		t.Fatalf("search institutions: %v", err)
	}

	if client.searchRequest.Query != "bank" {
		t.Fatalf("query = %q", client.searchRequest.Query)
	}
	if len(client.searchRequest.Products) != 1 || client.searchRequest.Products[0] != plaid.PRODUCTS_TRANSACTIONS {
		t.Fatalf("products = %#v", client.searchRequest.Products)
	}
	if len(client.searchRequest.CountryCodes) != 2 || client.searchRequest.CountryCodes[0] != plaid.COUNTRYCODE_US || client.searchRequest.CountryCodes[1] != plaid.COUNTRYCODE_CA {
		t.Fatalf("country codes = %#v", client.searchRequest.CountryCodes)
	}
	if len(institutions) != 1 || institutions[0].ID != "plaid:ins_123" || institutions[0].ProviderInstitutionID != "ins_123" {
		t.Fatalf("institutions = %#v", institutions)
	}
}

func TestPlaidProviderCreateLinkSessionRequiresExplicitProducts(t *testing.T) {
	provider := plaidProvider{
		cfg: config.ProviderConfig{Fields: map[string]string{
			"client_id":     "client-id",
			"secret":        "secret",
			"country_codes": "US",
		}},
		client: &fakePlaidLinkTokenClient{},
	}

	_, err := provider.CreateLinkSession(context.Background(), LinkRequest{State: "state-123"})
	if err == nil {
		t.Fatal("expected explicit products error")
	}
}

func TestPlaidProviderExchangeLinkTokenExchangesPublicTokenAndMapsLinkedItem(t *testing.T) {
	client := &fakePlaidLinkTokenClient{
		exchangeResult: PlaidPublicTokenExchangeResult{
			AccessToken: "access-token",
			ItemID:      "item_123",
		},
	}
	provider := plaidProvider{
		cfg: config.ProviderConfig{Fields: map[string]string{
			"client_id": "client-id",
			"secret":    "secret",
			"products":  "transactions,liabilities",
		}},
		client: client,
	}

	linked, err := provider.ExchangeLinkToken(context.Background(), LinkSession{
		Provider: "plaid",
		State:    "state-123",
	}, LinkCallback{
		PublicToken: "public-token",
		State:       "state-123",
		Metadata: LinkMetadata{
			Institution: LinkInstitutionMetadata{ID: "ins_123", Name: "Bank"},
		},
	})
	if err != nil {
		t.Fatalf("exchange link token: %v", err)
	}

	if client.publicToken != "public-token" {
		t.Fatalf("public token = %q", client.publicToken)
	}
	if linked.Institution.ID != "plaid:ins_123" || linked.Institution.ProviderInstitutionID != "ins_123" {
		t.Fatalf("institution = %#v", linked.Institution)
	}
	if linked.ProviderItem.ID != "plaid:item_123" || linked.ProviderItem.ProviderExternalItemID != "item_123" {
		t.Fatalf("provider item = %#v", linked.ProviderItem)
	}
	if string(linked.ProviderItem.EncryptedAccessToken) != "access-token" || linked.ProviderItem.ExternalUserID != "state-123" {
		t.Fatalf("provider item token/user = %#v", linked.ProviderItem)
	}
	if len(linked.ProviderItem.Products) != 2 || linked.ProviderItem.Products[0] != "transactions" || linked.ProviderItem.Products[1] != "liabilities" {
		t.Fatalf("products = %#v", linked.ProviderItem.Products)
	}
}

func TestPlaidProviderExchangeLinkTokenRejectsStateMismatch(t *testing.T) {
	provider := plaidProvider{
		cfg:    config.ProviderConfig{Fields: map[string]string{"client_id": "client-id", "secret": "secret"}},
		client: &fakePlaidLinkTokenClient{},
	}

	_, err := provider.ExchangeLinkToken(context.Background(), LinkSession{State: "state-123"}, LinkCallback{
		PublicToken: "public-token",
		State:       "wrong",
		Metadata:    LinkMetadata{Institution: LinkInstitutionMetadata{ID: "ins_123", Name: "Bank"}},
	})
	if err == nil {
		t.Fatal("expected state mismatch error")
	}
}

func TestPlaidProviderSyncAccountsThenTransactionsWithCursor(t *testing.T) {
	current := nullablePlaidFloat(12.34)
	available := nullablePlaidFloat(10.00)
	currency := nullablePlaidString("USD")
	subtype := nullablePlaidAccountSubtype(plaid.ACCOUNTSUBTYPE_CHECKING)
	client := &fakePlaidLinkTokenClient{
		accounts: []plaid.AccountBase{{
			AccountId: "acc_1",
			Balances: plaid.AccountBalance{
				Current:         current,
				Available:       available,
				IsoCurrencyCode: currency,
			},
			Name:    "Checking",
			Type:    plaid.ACCOUNTTYPE_DEPOSITORY,
			Subtype: subtype,
		}},
		transactionPages: []plaid.TransactionsSyncResponse{{
			Added: []plaid.Transaction{{
				AccountId:       "acc_1",
				Amount:          6.25,
				IsoCurrencyCode: currency,
				Date:            "2026-05-10",
				Name:            "Coffee",
				MerchantName:    nullablePlaidString("Coffee Shop"),
				Pending:         false,
				TransactionId:   "tx_added",
				Category:        []string{"Food and Drink", "Coffee Shop"},
			}},
			Modified: []plaid.Transaction{{
				AccountId:       "acc_1",
				Amount:          -20,
				IsoCurrencyCode: currency,
				Date:            "2026-05-11",
				Name:            "Refund",
				Pending:         false,
				TransactionId:   "tx_modified",
			}},
			Removed:    []plaid.RemovedTransaction{{TransactionId: "tx_removed", AccountId: "acc_1"}},
			NextCursor: "cursor-next",
			HasMore:    false,
		}},
	}
	provider := plaidProvider{
		cfg:    config.ProviderConfig{Fields: map[string]string{"client_id": "client-id", "secret": "secret"}},
		client: client,
	}
	sink := &recordingSyncSink{}

	result, err := provider.Sync(context.Background(), ProviderItem{
		ID:                   "pi_1",
		EncryptedAccessToken: []byte("access-token"),
		TransactionCursor:    "cursor-old",
	}, sink)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	if client.accountsAccessToken != "access-token" {
		t.Fatalf("accounts access token = %q", client.accountsAccessToken)
	}
	if len(client.transactionRequests) != 1 || client.transactionRequests[0].cursor != "cursor-old" {
		t.Fatalf("transaction requests = %#v", client.transactionRequests)
	}
	if len(sink.calls) != 4 || sink.calls[0] != "account:acc_1" || sink.calls[1] != "transaction:tx_added" || sink.calls[2] != "transaction:tx_modified" || sink.calls[3] != "removed:tx_removed" {
		t.Fatalf("calls = %#v", sink.calls)
	}
	if sink.accounts[0].ProviderItemID != "pi_1" || sink.accounts[0].CurrentBalanceMinorUnits != 1234 {
		t.Fatalf("account = %#v", sink.accounts[0])
	}
	if sink.transactions[0].AmountMinorUnits != -625 || sink.transactions[1].AmountMinorUnits != 2000 {
		t.Fatalf("transactions = %#v", sink.transactions)
	}
	if result.AccountsSeen != 1 || result.TransactionsAdded != 1 || result.TransactionsModified != 1 || result.TransactionsRemoved != 1 || result.NextTransactionCursor != "cursor-next" {
		t.Fatalf("result = %#v", result)
	}
}

type fakePlaidLinkTokenClient struct {
	request             plaid.LinkTokenCreateRequest
	searchRequest       plaid.InstitutionsSearchRequest
	linkToken           string
	publicToken         string
	exchangeResult      PlaidPublicTokenExchangeResult
	institutions        []plaid.Institution
	accountsAccessToken string
	accounts            []plaid.AccountBase
	transactionRequests []plaidTransactionRequest
	transactionPages    []plaid.TransactionsSyncResponse
}

func (c *fakePlaidLinkTokenClient) CreateLinkToken(ctx context.Context, request plaid.LinkTokenCreateRequest) (string, error) {
	c.request = request
	return c.linkToken, nil
}

func (c *fakePlaidLinkTokenClient) ExchangePublicToken(ctx context.Context, publicToken string) (PlaidPublicTokenExchangeResult, error) {
	c.publicToken = publicToken
	return c.exchangeResult, nil
}

func (c *fakePlaidLinkTokenClient) SearchInstitutions(ctx context.Context, request plaid.InstitutionsSearchRequest) ([]plaid.Institution, error) {
	c.searchRequest = request
	return c.institutions, nil
}

func (c *fakePlaidLinkTokenClient) GetAccounts(ctx context.Context, accessToken string) ([]plaid.AccountBase, error) {
	c.accountsAccessToken = accessToken
	return c.accounts, nil
}

func (c *fakePlaidLinkTokenClient) SyncTransactions(ctx context.Context, accessToken string, cursor string) (plaid.TransactionsSyncResponse, error) {
	c.transactionRequests = append(c.transactionRequests, plaidTransactionRequest{accessToken: accessToken, cursor: cursor})
	index := len(c.transactionRequests) - 1
	if index >= len(c.transactionPages) {
		return plaid.TransactionsSyncResponse{}, nil
	}
	return c.transactionPages[index], nil
}

type plaidTransactionRequest struct {
	accessToken string
	cursor      string
}

type recordingSyncSink struct {
	calls        []string
	accounts     []FinancialAccount
	transactions []Transaction
}

func (s *recordingSyncSink) UpsertInstitution(ctx context.Context, institution Institution) error {
	return nil
}
func (s *recordingSyncSink) UpsertProviderItem(ctx context.Context, item ProviderItem) error {
	return nil
}
func (s *recordingSyncSink) UpsertAccount(ctx context.Context, account FinancialAccount) error {
	s.calls = append(s.calls, "account:"+account.ProviderAccountID)
	s.accounts = append(s.accounts, account)
	return nil
}
func (s *recordingSyncSink) UpsertTransaction(ctx context.Context, transaction Transaction) error {
	s.calls = append(s.calls, "transaction:"+transaction.ProviderTransactionID)
	s.transactions = append(s.transactions, transaction)
	return nil
}
func (s *recordingSyncSink) UpsertRecurring(ctx context.Context, recurring Recurring) error {
	return nil
}
func (s *recordingSyncSink) MarkTransactionRemoved(ctx context.Context, providerItemID string, providerTransactionID string) error {
	s.calls = append(s.calls, "removed:"+providerTransactionID)
	return nil
}
func (s *recordingSyncSink) RecordSyncRun(ctx context.Context, run SyncRun) error { return nil }

func nullablePlaidString(value string) plaid.NullableString {
	var nullable plaid.NullableString
	nullable.Set(&value)
	return nullable
}

func nullablePlaidFloat(value float64) plaid.NullableFloat64 {
	var nullable plaid.NullableFloat64
	nullable.Set(&value)
	return nullable
}

func nullablePlaidAccountSubtype(value plaid.AccountSubtype) plaid.NullableAccountSubtype {
	var nullable plaid.NullableAccountSubtype
	nullable.Set(&value)
	return nullable
}
