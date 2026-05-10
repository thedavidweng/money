package syncer

import (
	"context"
	"errors"
	"testing"

	"github.com/thedavidweng/money/internal/providers"
	"github.com/thedavidweng/money/internal/store"
)

func TestSyncReturnsWarningWhenNoProviderItems(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()

	result, err := Sync(ctx, db, fakeRegistry{}, Options{})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(result.Items) != 0 || len(result.Warnings) != 1 || result.Warnings[0].Code != "NO_LINKED_PROVIDER_ITEMS" {
		t.Fatalf("result = %#v", result)
	}
}

func TestSyncDispatchesLinkedItemToProviderAndAdvancesCursor(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()
	if err := db.StoreLinkedProviderItem(ctx, store.LinkedProviderItem{
		Institution: store.LinkedInstitution{ID: "inst_sync", Name: "Sync Bank", Provider: "plaid", ProviderInstitutionID: "ins_sync"},
		Item: store.LinkedItem{
			ID:                     "pi_sync",
			Provider:               "plaid",
			InstitutionID:          "inst_sync",
			ProviderExternalItemID: "item_sync",
			EncryptedAccessToken:   []byte("token"),
			Status:                 "active",
			Products:               []string{"transactions"},
		},
	}); err != nil {
		t.Fatalf("store linked item: %v", err)
	}
	provider := &fakeSyncProvider{result: providers.SyncResult{
		Provider:              "plaid",
		ProviderItemID:        "pi_sync",
		AccountsSeen:          1,
		TransactionsAdded:     1,
		NextTransactionCursor: "cursor-next",
	}}

	result, err := Sync(ctx, db, fakeRegistry{provider: provider}, Options{Provider: "plaid"})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if provider.item.ID != "pi_sync" || string(provider.item.EncryptedAccessToken) != "token" {
		t.Fatalf("provider item = %#v", provider.item)
	}
	if len(result.Items) != 1 || result.Items[0].NextTransactionCursor != "cursor-next" {
		t.Fatalf("result = %#v", result)
	}
	item, err := db.GetProviderItem(ctx, "pi_sync")
	if err != nil {
		t.Fatalf("get provider item: %v", err)
	}
	if item.TransactionCursor != "cursor-next" {
		t.Fatalf("cursor = %q", item.TransactionCursor)
	}
}

func TestSyncPartialFailurePreservesPerItemResults(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()
	for _, item := range []store.LinkedItem{
		{ID: "pi_ok", Provider: "plaid", InstitutionID: "inst_1", ProviderExternalItemID: "item_ok", EncryptedAccessToken: []byte("token"), Status: "active"},
		{ID: "pi_bad", Provider: "plaid", InstitutionID: "inst_1", ProviderExternalItemID: "item_bad", EncryptedAccessToken: []byte("token"), Status: "active"},
	} {
		if err := db.StoreLinkedProviderItem(ctx, store.LinkedProviderItem{
			Institution: store.LinkedInstitution{ID: item.InstitutionID, Name: "Bank", Provider: "plaid", ProviderInstitutionID: "ins_1"},
			Item:        item,
		}); err != nil {
			t.Fatalf("store linked item: %v", err)
		}
	}
	provider := &fakeSyncProvider{
		result: providers.SyncResult{Provider: "plaid", AccountsSeen: 1},
		failByItem: map[string]error{
			"pi_bad": providers.ProviderAPIError{Provider: "plaid", StatusCode: 500, Code: "PLAID_DOWN", Message: "Plaid unavailable"},
		},
	}

	result, err := Sync(ctx, db, fakeRegistry{provider: provider}, Options{Provider: "plaid"})
	var partial PartialFailure
	if !errors.As(err, &partial) {
		t.Fatalf("error = %#v, want PartialFailure", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("items = %#v", result.Items)
	}
	byID := map[string]ItemResult{}
	for _, item := range result.Items {
		byID[item.ProviderItemID] = item
	}
	if byID["pi_ok"].Status != "ok" || byID["pi_bad"].Status != "error" || byID["pi_bad"].ErrorCode != "PLAID_DOWN" {
		t.Fatalf("result = %#v", result)
	}
}

func TestSyncPartialFailureCoversPlaidAndBridgeItems(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer db.Close()
	for _, linked := range []store.LinkedProviderItem{
		{
			Institution: store.LinkedInstitution{ID: "inst_plaid", Name: "Plaid Bank", Provider: "plaid", ProviderInstitutionID: "ins_plaid"},
			Item:        store.LinkedItem{ID: "pi_plaid", Provider: "plaid", InstitutionID: "inst_plaid", ProviderExternalItemID: "item_plaid", EncryptedAccessToken: []byte("token"), Status: "active"},
		},
		{
			Institution: store.LinkedInstitution{ID: "inst_bridge", Name: "Bridge Bank", Provider: "bridge", ProviderInstitutionID: "bank_bridge"},
			Item:        store.LinkedItem{ID: "pi_bridge", Provider: "bridge", InstitutionID: "inst_bridge", ProviderExternalItemID: "item_bridge", EncryptedAccessToken: []byte("bridge-user"), Status: "active"},
		},
	} {
		if err := db.StoreLinkedProviderItem(ctx, linked); err != nil {
			t.Fatalf("store linked item: %v", err)
		}
	}

	result, err := Sync(ctx, db, fakeRegistry{providers: map[string]providers.Provider{
		"plaid":  &fakeSyncProvider{name: "plaid", result: providers.SyncResult{Provider: "plaid", AccountsSeen: 1}},
		"bridge": &fakeSyncProvider{name: "bridge", failByItem: map[string]error{"pi_bridge": providers.ProviderAPIError{Provider: "bridge", StatusCode: 401, Code: "UNAUTHORIZED", Message: "Bridge authorization failed"}}},
	}}, Options{})
	var partial PartialFailure
	if !errors.As(err, &partial) {
		t.Fatalf("error = %#v, want PartialFailure", err)
	}
	byID := map[string]ItemResult{}
	for _, item := range result.Items {
		byID[item.ProviderItemID] = item
	}
	if byID["pi_plaid"].Status != "ok" || byID["pi_bridge"].Status != "error" || byID["pi_bridge"].ErrorCode != "UNAUTHORIZED" {
		t.Fatalf("result = %#v", result)
	}
}

type fakeRegistry struct {
	provider  providers.Provider
	providers map[string]providers.Provider
}

func (r fakeRegistry) Get(name string) (providers.Provider, bool) {
	if r.providers != nil {
		provider, ok := r.providers[name]
		return provider, ok
	}
	if r.provider == nil || r.provider.Name() != name {
		return nil, false
	}
	return r.provider, true
}

type fakeSyncProvider struct {
	name       string
	item       providers.ProviderItem
	result     providers.SyncResult
	failByItem map[string]error
}

func (p *fakeSyncProvider) Name() string {
	if p.name == "" {
		return "plaid"
	}
	return p.name
}
func (p *fakeSyncProvider) ValidateConfig(ctx context.Context) []providers.ConfigDiagnostic {
	return nil
}
func (p *fakeSyncProvider) SearchInstitutions(ctx context.Context, query string) ([]providers.Institution, error) {
	return nil, nil
}
func (p *fakeSyncProvider) CreateLinkSession(ctx context.Context, request providers.LinkRequest) (providers.LinkSession, error) {
	return providers.LinkSession{}, nil
}
func (p *fakeSyncProvider) ExchangeLinkToken(ctx context.Context, session providers.LinkSession, callback providers.LinkCallback) (providers.LinkedItem, error) {
	return providers.LinkedItem{}, nil
}
func (p *fakeSyncProvider) Sync(ctx context.Context, item providers.ProviderItem, sink providers.SyncSink) (providers.SyncResult, error) {
	p.item = item
	if err := p.failByItem[item.ID]; err != nil {
		return providers.SyncResult{}, err
	}
	return p.result, nil
}
