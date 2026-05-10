package linking

import (
	"context"

	"github.com/thedavidweng/money/internal/providers"
	"github.com/thedavidweng/money/internal/store"
)

type Store interface {
	StoreLinkedProviderItem(ctx context.Context, linked store.LinkedProviderItem) error
}

type LinkResult struct {
	Provider       string `json:"provider"`
	ProviderItemID string `json:"provider_item_id"`
	InstitutionID  string `json:"institution_id"`
}

func CompleteProviderLink(ctx context.Context, target Store, provider providers.Provider, session providers.LinkSession, callback providers.LinkCallback) (LinkResult, error) {
	linked, err := provider.ExchangeLinkToken(ctx, session, callback)
	if err != nil {
		return LinkResult{}, err
	}
	if err := target.StoreLinkedProviderItem(ctx, store.LinkedProviderItem{
		Institution: store.LinkedInstitution{
			ID:                    linked.Institution.ID,
			Name:                  linked.Institution.Name,
			Provider:              linked.Institution.Provider,
			ProviderInstitutionID: linked.Institution.ProviderInstitutionID,
		},
		Item: store.LinkedItem{
			ID:                     linked.ProviderItem.ID,
			Provider:               linked.ProviderItem.Provider,
			InstitutionID:          linked.ProviderItem.InstitutionID,
			ProviderExternalItemID: linked.ProviderItem.ProviderExternalItemID,
			EncryptedAccessToken:   linked.ProviderItem.EncryptedAccessToken,
			Status:                 linked.ProviderItem.Status,
			TransactionCursor:      linked.ProviderItem.TransactionCursor,
			ExternalUserID:         linked.ProviderItem.ExternalUserID,
			Products:               linked.ProviderItem.Products,
		},
	}); err != nil {
		return LinkResult{}, err
	}
	return LinkResult{
		Provider:       linked.ProviderItem.Provider,
		ProviderItemID: linked.ProviderItem.ID,
		InstitutionID:  linked.Institution.ID,
	}, nil
}
