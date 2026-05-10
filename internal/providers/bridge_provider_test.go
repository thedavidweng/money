package providers

import (
	"context"
	"testing"
	"time"

	"github.com/thedavidweng/money/internal/config"
)

func TestBridgeProviderCreateLinkSessionCreatesUserTokenAndConnectSession(t *testing.T) {
	client := &fakeBridgeAPI{
		authToken: BridgeAuthToken{AccessToken: "bridge-user-token", ExternalUserID: "bridge-state"},
		connect:   BridgeConnectSession{ID: "session-1", URL: "https://connect.bridgeapi.io/session/session-1"},
		existingItems: []BridgeItem{
			{ID: "item_existing", ProviderID: "bank_1", ProviderName: "Bank", Status: "0"},
		},
	}
	provider := bridgeProvider{
		cfg: config.ProviderConfig{Fields: map[string]string{
			"client_id":     "client-id",
			"client_secret": "secret",
			"user_email":    "user@example.test",
			"country_code":  "FR",
			"capabilities":  "ais",
		}},
		client:       client,
		pollInterval: time.Millisecond,
	}

	session, err := provider.CreateLinkSession(context.Background(), LinkRequest{State: "state"})
	if err != nil {
		t.Fatalf("create link session: %v", err)
	}

	if client.createdExternalUserID != "bridge-state" {
		t.Fatalf("created external user id = %q", client.createdExternalUserID)
	}
	if client.authExternalUserID != "bridge-state" {
		t.Fatalf("auth external user id = %q", client.authExternalUserID)
	}
	if client.connectRequest.UserEmail != "user@example.test" || client.connectRequest.CountryCode != "FR" {
		t.Fatalf("connect request = %#v", client.connectRequest)
	}
	if session.URL != "https://connect.bridgeapi.io/session/session-1" || session.ProviderAccessToken != "bridge-user-token" {
		t.Fatalf("session = %#v", session)
	}
	if len(session.ExistingProviderItemIDs) != 1 || session.ExistingProviderItemIDs[0] != "item_existing" {
		t.Fatalf("existing ids = %#v", session.ExistingProviderItemIDs)
	}
}

func TestBridgeProviderExchangeLinkTokenPollsNewItemAndMapsLinkedItem(t *testing.T) {
	client := &fakeBridgeAPI{
		polledItems: [][]BridgeItem{
			{{ID: "item_existing", ProviderID: "bank_1", ProviderName: "Bank", Status: "0"}},
			{{ID: "item_existing", ProviderID: "bank_1", ProviderName: "Bank", Status: "0"}, {ID: "item_new", ProviderID: "bank_2", ProviderName: "New Bank", Status: "0"}},
		},
	}
	provider := bridgeProvider{
		cfg: config.ProviderConfig{Fields: map[string]string{
			"client_id":     "client-id",
			"client_secret": "secret",
			"user_email":    "user@example.test",
			"capabilities":  "ais",
		}},
		client:       client,
		pollInterval: time.Millisecond,
	}

	linked, err := provider.ExchangeLinkToken(context.Background(), LinkSession{
		Provider:                "bridge",
		State:                   "bridge-user",
		ProviderAccessToken:     "bridge-user-token",
		ExistingProviderItemIDs: []string{"item_existing"},
	}, LinkCallback{})
	if err != nil {
		t.Fatalf("exchange link token: %v", err)
	}

	if linked.Institution.ID != "bridge:bank_2" || linked.Institution.Name != "New Bank" {
		t.Fatalf("institution = %#v", linked.Institution)
	}
	if linked.ProviderItem.ID != "bridge:item_new" || linked.ProviderItem.ExternalUserID != "bridge-user" {
		t.Fatalf("provider item = %#v", linked.ProviderItem)
	}
	if string(linked.ProviderItem.EncryptedAccessToken) != "bridge-user" {
		t.Fatalf("encrypted credential = %q", string(linked.ProviderItem.EncryptedAccessToken))
	}
}

func TestBridgeProviderSyncAccountsThenTransactionsWithUpdatedAtCursor(t *testing.T) {
	client := &fakeBridgeAPI{
		authToken: BridgeAuthToken{AccessToken: "bridge-access", ExternalUserID: "bridge-user"},
		accounts: []BridgeAccount{{
			ID:           "acc_1",
			ItemID:       "item_1",
			Name:         "Compte Courant",
			Type:         "checking",
			Balance:      123.45,
			Currency:     "EUR",
			DataAccess:   "enabled",
			UpdatedAt:    "2026-05-10T10:00:00Z",
			ProviderID:   "bank_1",
			ProviderName: "Bridge Bank",
		}},
		transactions: []BridgeSyncTransaction{
			{
				ID:          "tx_1",
				AccountID:   "acc_1",
				Date:        "2026-05-10",
				UpdatedAt:   "2026-05-10T11:00:00Z",
				Amount:      -12.34,
				Description: "Lunch",
				Currency:    "EUR",
			},
			{
				ID:        "tx_deleted",
				AccountID: "acc_1",
				UpdatedAt: "2026-05-10T12:00:00Z",
				Deleted:   true,
			},
		},
	}
	provider := bridgeProvider{
		cfg:          config.ProviderConfig{Fields: map[string]string{"client_id": "client-id", "client_secret": "secret"}},
		client:       client,
		pollInterval: time.Millisecond,
	}
	sink := &recordingSyncSink{}

	result, err := provider.Sync(context.Background(), ProviderItem{
		ID:                     "bridge:item_1",
		ProviderExternalItemID: "item_1",
		EncryptedAccessToken:   []byte("bridge-user"),
		ExternalUserID:         "bridge-user",
		TransactionCursor:      "2026-05-09T00:00:00Z",
	}, sink)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	if client.authExternalUserID != "bridge-user" {
		t.Fatalf("auth external user id = %q", client.authExternalUserID)
	}
	if client.accountsItemID != "item_1" {
		t.Fatalf("accounts item id = %q", client.accountsItemID)
	}
	if client.transactionsSince != "2026-05-09T00:00:00Z" {
		t.Fatalf("transactions since = %q", client.transactionsSince)
	}
	if len(sink.calls) != 3 || sink.calls[0] != "account:acc_1" || sink.calls[1] != "transaction:tx_1" || sink.calls[2] != "removed:tx_deleted" {
		t.Fatalf("calls = %#v", sink.calls)
	}
	if sink.accounts[0].Type != "depository" || sink.accounts[0].CurrentBalanceMinorUnits != 12345 {
		t.Fatalf("account = %#v", sink.accounts[0])
	}
	if sink.transactions[0].AmountMinorUnits != -1234 {
		t.Fatalf("transaction = %#v", sink.transactions[0])
	}
	if result.AccountsSeen != 1 || result.TransactionsModified != 1 || result.TransactionsRemoved != 1 || result.NextTransactionCursor != "2026-05-10T12:00:00Z" {
		t.Fatalf("result = %#v", result)
	}
}

type fakeBridgeAPI struct {
	createdExternalUserID string
	authExternalUserID    string
	authToken             BridgeAuthToken
	connectRequest        BridgeConnectSessionRequest
	connect               BridgeConnectSession
	existingItems         []BridgeItem
	polledItems           [][]BridgeItem
	pollCount             int
	accountsItemID        string
	accounts              []BridgeAccount
	transactionsSince     string
	transactions          []BridgeSyncTransaction
}

func (c *fakeBridgeAPI) CreateBridgeUser(ctx context.Context, externalUserID string) error {
	c.createdExternalUserID = externalUserID
	return nil
}

func (c *fakeBridgeAPI) CreateBridgeAuthToken(ctx context.Context, externalUserID string) (BridgeAuthToken, error) {
	c.authExternalUserID = externalUserID
	token := c.authToken
	if token.ExternalUserID == "" {
		token.ExternalUserID = externalUserID
	}
	return token, nil
}

func (c *fakeBridgeAPI) CreateBridgeConnectSession(ctx context.Context, accessToken string, request BridgeConnectSessionRequest) (BridgeConnectSession, error) {
	c.connectRequest = request
	return c.connect, nil
}

func (c *fakeBridgeAPI) ListBridgeItems(ctx context.Context, accessToken string) ([]BridgeItem, error) {
	if len(c.polledItems) == 0 {
		return c.existingItems, nil
	}
	index := c.pollCount
	if index >= len(c.polledItems) {
		index = len(c.polledItems) - 1
	}
	c.pollCount++
	return c.polledItems[index], nil
}

func (c *fakeBridgeAPI) ListBridgeAccounts(ctx context.Context, accessToken string, itemID string) ([]BridgeAccount, error) {
	c.accountsItemID = itemID
	return c.accounts, nil
}

func (c *fakeBridgeAPI) ListBridgeTransactions(ctx context.Context, accessToken string, since string) ([]BridgeSyncTransaction, error) {
	c.transactionsSince = since
	return c.transactions, nil
}
