package linking

import (
	"context"
	"errors"
	"strings"
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
	defer func() { _ = db.Close() }()

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
	defer func() { _ = db.Close() }()

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

func TestCompleteProviderLinkDoesNotExchangeTokenForCancelOrError(t *testing.T) {
	ctx := context.Background()
	db, err := store.OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	_, err = CompleteProviderLink(ctx, db, exchangeCountingProvider{}, providers.LinkSession{Provider: "plaid", State: "state"}, providers.LinkCallback{
		Status: "cancel",
		State:  "state",
	})
	var canceled LinkCanceledError
	if !errors.As(err, &canceled) {
		t.Fatalf("cancel err = %#v", err)
	}

	_, err = CompleteProviderLink(ctx, db, exchangeCountingProvider{}, providers.LinkSession{Provider: "plaid", State: "state"}, providers.LinkCallback{
		Status: "error",
		State:  "state",
		Error:  providers.LinkError{Type: "ITEM_ERROR", Code: "INVALID_CREDENTIALS", Message: "bad credentials"},
		Metadata: providers.LinkMetadata{
			RequestID:     "req_123",
			LinkSessionID: "link-session",
		},
	})
	var linkErr LinkFlowError
	if !errors.As(err, &linkErr) || linkErr.Code != "INVALID_CREDENTIALS" {
		t.Fatalf("link err = %#v", err)
	}
	message := err.Error()
	for _, want := range []string{"bad credentials", "ITEM_ERROR", "INVALID_CREDENTIALS", "req_123", "link-session"} {
		if !strings.Contains(message, want) {
			t.Fatalf("link error message %q missing %q", message, want)
		}
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

type exchangeCountingProvider struct {
	fakeProvider
}

func (exchangeCountingProvider) ExchangeLinkToken(ctx context.Context, session providers.LinkSession, callback providers.LinkCallback) (providers.LinkedItem, error) {
	panic("ExchangeLinkToken should not be called for canceled or errored Link callbacks")
}
