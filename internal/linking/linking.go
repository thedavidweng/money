package linking

import (
	"context"
	"fmt"
	"strings"

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
	switch callback.Status {
	case "", "success":
	case "cancel":
		return LinkResult{}, LinkCanceledError{LinkSessionID: callback.Metadata.LinkSessionID}
	case "error":
		return LinkResult{}, LinkFlowError{
			Type:          callback.Error.Type,
			Code:          callback.Error.Code,
			Message:       callback.Error.Message,
			RequestID:     callback.Metadata.RequestID,
			LinkSessionID: callback.Metadata.LinkSessionID,
		}
	default:
		return LinkResult{}, fmt.Errorf("unsupported Plaid Link callback status %q", callback.Status)
	}
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

type LinkCanceledError struct {
	LinkSessionID string
}

func (e LinkCanceledError) Error() string {
	if e.LinkSessionID == "" {
		return "Plaid Link was canceled"
	}
	return "Plaid Link was canceled for session " + e.LinkSessionID
}

type LinkFlowError struct {
	Type          string
	Code          string
	Message       string
	RequestID     string
	LinkSessionID string
}

func (e LinkFlowError) Error() string {
	message := e.Message
	if message == "" {
		message = "Plaid Link returned an error"
	}
	var details []string
	if e.Type != "" {
		details = append(details, "type="+e.Type)
	}
	if e.Code != "" {
		details = append(details, "code="+e.Code)
	}
	if e.RequestID != "" {
		details = append(details, "request_id="+e.RequestID)
	}
	if e.LinkSessionID != "" {
		details = append(details, "link_session_id="+e.LinkSessionID)
	}
	if len(details) > 0 {
		message += " (" + strings.Join(details, ", ") + ")"
	}
	return message
}
