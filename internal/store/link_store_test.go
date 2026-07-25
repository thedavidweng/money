package store

import (
	"context"
	"testing"
)

func TestStoreLinkedProviderItemPersistsInstitutionAndTokenOnlyInStore(t *testing.T) {
	ctx := context.Background()
	db, err := OpenDemo(ctx)
	if err != nil {
		t.Fatalf("open demo: %v", err)
	}
	defer func() { _ = db.Close() }()

	err = db.StoreLinkedProviderItem(ctx, &LinkedProviderItem{
		Institution: LinkedInstitution{
			ID:                    "inst_test_bank",
			Name:                  "Test Bank",
			Provider:              "plaid",
			ProviderInstitutionID: "ins_test",
		},
		Item: LinkedItem{
			ID:                     "pi_test_item",
			Provider:               "plaid",
			InstitutionID:          "inst_test_bank",
			ProviderExternalItemID: "item_external",
			AccessToken:            "access-token-secret",
			Status:                 "active",
			Products:               []string{"transactions"},
			TransactionCursor:      "cursor",
		},
	})
	if err != nil {
		t.Fatalf("store linked provider item: %v", err)
	}

	item, err := db.GetProviderItem(ctx, "pi_test_item")
	if err != nil {
		t.Fatalf("get provider item: %v", err)
	}
	if item.Provider != "plaid" || item.ProviderExternalItemID != "item_external" {
		t.Fatalf("item = %#v", item)
	}
	if string(item.EncryptedAccessToken) != "access-token-secret" {
		t.Fatalf("access token was not persisted in provider_items")
	}
}
