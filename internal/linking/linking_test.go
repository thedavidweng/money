package linking

import (
	"context"
	"testing"

	"github.com/thedavidweng/money/internal/providers"
	"github.com/thedavidweng/money/internal/store"
)

func TestCompleteProviderLinkExchangesTokenAndStoresProviderItem(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer db.Close()

	result, err := CompleteProviderLink(ctx, db, fakeProvider{}, providers.LinkSession{
		Provider: "plaid",
		URL:      "http://127.0.0.1/link",
		State:    "state",
	}, providers.LinkCallback{
		PublicToken: "public-token",
		State:       "state",
	})
	if err != nil {
		t.Fatalf("complete provider link: %v", err)
	}
	if result.ProviderItemID != "pi_fake" {
		t.Fatalf("provider item id = %q", result.ProviderItemID)
	}

	item, err := db.GetProviderItem(ctx, "pi_fake")
	if err != nil {
		t.Fatalf("provider item not stored: %v", err)
	}
	if item.ProviderExternalItemID != "item_fake" || string(item.EncryptedAccessToken) != "access-token" {
		t.Fatalf("stored item = %#v", item)
	}
}

func TestCompleteProviderLinkStoresTokenInEncryptedStore(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenEncrypted(ctx, t.TempDir()+"/money.db", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("open encrypted store: %v", err)
	}
	defer db.Close()

	result, err := CompleteProviderLink(ctx, db, fakeProvider{}, providers.LinkSession{
		Provider: "plaid",
		State:    "state",
	}, providers.LinkCallback{
		PublicToken: "public-token",
		State:       "state",
	})
	if err != nil {
		t.Fatalf("complete provider link: %v", err)
	}

	item, err := db.GetProviderItem(ctx, result.ProviderItemID)
	if err != nil {
		t.Fatalf("provider item not stored: %v", err)
	}
	if string(item.EncryptedAccessToken) != "access-token" {
		t.Fatalf("access token = %q", string(item.EncryptedAccessToken))
	}
}

type fakeProvider struct{}

func (fakeProvider) Name() string                                                    { return "plaid" }
func (fakeProvider) ValidateConfig(ctx context.Context) []providers.ConfigDiagnostic { return nil }
func (fakeProvider) SearchInstitutions(ctx context.Context, query string) ([]providers.Institution, error) {
	return nil, nil
}
func (fakeProvider) CreateLinkSession(ctx context.Context, request providers.LinkRequest) (providers.LinkSession, error) {
	return providers.LinkSession{}, nil
}
func (fakeProvider) ExchangeLinkToken(ctx context.Context, session providers.LinkSession, callback providers.LinkCallback) (providers.LinkedItem, error) {
	return providers.LinkedItem{
		Institution: providers.Institution{
			ID:                    "inst_fake",
			Name:                  "Fake Bank",
			Provider:              "plaid",
			ProviderInstitutionID: "ins_fake",
		},
		ProviderItem: providers.ProviderItem{
			ID:                     "pi_fake",
			Provider:               "plaid",
			InstitutionID:          "inst_fake",
			ProviderExternalItemID: "item_fake",
			EncryptedAccessToken:   []byte("access-token"),
			Status:                 "active",
			TransactionCursor:      "cursor",
		},
	}, nil
}
func (fakeProvider) Sync(ctx context.Context, item providers.ProviderItem, sink providers.SyncSink) (providers.SyncResult, error) {
	return providers.SyncResult{}, nil
}
