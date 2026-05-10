package syncer

import (
	"context"
	"fmt"
	"time"

	"github.com/thedavidweng/money/internal/providers"
	"github.com/thedavidweng/money/internal/store"
)

type Store interface {
	ListProviderItems(ctx context.Context, query store.ProviderItemQuery) ([]store.LinkedItem, error)
	providers.SyncSink
}

type Registry interface {
	Get(name string) (providers.Provider, bool)
}

type Options struct {
	Provider       string
	ProviderItemID string
}

type Result struct {
	Items    []ItemResult `json:"items"`
	Warnings []Warning    `json:"warnings"`
}

type ItemResult struct {
	Provider              string `json:"provider"`
	ProviderItemID        string `json:"provider_item_id"`
	Status                string `json:"status"`
	AccountsSeen          int    `json:"accounts_seen"`
	TransactionsAdded     int    `json:"transactions_added"`
	TransactionsModified  int    `json:"transactions_modified"`
	TransactionsRemoved   int    `json:"transactions_removed"`
	RecurringStreamsSeen  int    `json:"recurring_streams_seen"`
	NextTransactionCursor string `json:"next_transaction_cursor,omitempty"`
	ErrorCode             string `json:"error_code,omitempty"`
	ErrorMessage          string `json:"error_message,omitempty"`
}

type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PartialFailure struct {
	Result Result
}

func (e PartialFailure) Error() string {
	return "one or more provider items failed to sync"
}

func Sync(ctx context.Context, target Store, registry Registry, options Options) (Result, error) {
	linkedItems, err := target.ListProviderItems(ctx, store.ProviderItemQuery{Provider: options.Provider, ProviderItemID: options.ProviderItemID})
	if err != nil {
		return Result{}, err
	}
	if len(linkedItems) == 0 {
		return Result{Warnings: []Warning{{Code: "NO_LINKED_PROVIDER_ITEMS", Message: "No linked Provider Items found."}}}, nil
	}

	result := Result{Items: []ItemResult{}, Warnings: []Warning{}}
	var failed bool
	for _, linked := range linkedItems {
		itemResult := syncOne(ctx, target, registry, linked)
		if itemResult.Status == "error" {
			failed = true
		}
		result.Items = append(result.Items, itemResult)
	}
	if failed {
		return result, PartialFailure{Result: result}
	}
	return result, nil
}

func syncOne(ctx context.Context, target Store, registry Registry, linked store.LinkedItem) ItemResult {
	started := time.Now().UTC().Format(time.RFC3339)
	itemResult := ItemResult{Provider: linked.Provider, ProviderItemID: linked.ID, Status: "ok"}
	provider, ok := registry.Get(linked.Provider)
	if !ok {
		itemResult.Status = "error"
		itemResult.ErrorCode = "PROVIDER_NOT_REGISTERED"
		itemResult.ErrorMessage = fmt.Sprintf("provider %q is not registered", linked.Provider)
		recordSyncRun(ctx, target, itemResult, started)
		return itemResult
	}
	syncResult, err := provider.Sync(ctx, providers.ProviderItem{
		ID:                     linked.ID,
		Provider:               linked.Provider,
		InstitutionID:          linked.InstitutionID,
		ProviderExternalItemID: linked.ProviderExternalItemID,
		EncryptedAccessToken:   linked.EncryptedAccessToken,
		TransactionCursor:      linked.TransactionCursor,
		ExternalUserID:         linked.ExternalUserID,
		Status:                 linked.Status,
		Products:               linked.Products,
	}, target)
	if err != nil {
		classified := providers.ClassifyProviderError(linked.Provider, err)
		itemResult.Status = "error"
		itemResult.ErrorCode = classified.Code
		itemResult.ErrorMessage = classified.Message
		recordSyncRun(ctx, target, itemResult, started)
		return itemResult
	}
	itemResult.AccountsSeen = syncResult.AccountsSeen
	itemResult.TransactionsAdded = syncResult.TransactionsAdded
	itemResult.TransactionsModified = syncResult.TransactionsModified
	itemResult.TransactionsRemoved = syncResult.TransactionsRemoved
	itemResult.RecurringStreamsSeen = syncResult.RecurringStreamsSeen
	itemResult.NextTransactionCursor = syncResult.NextTransactionCursor
	if syncResult.NextTransactionCursor != "" {
		if err := target.UpsertProviderItem(ctx, providers.ProviderItem{
			ID:                     linked.ID,
			Provider:               linked.Provider,
			InstitutionID:          linked.InstitutionID,
			ProviderExternalItemID: linked.ProviderExternalItemID,
			EncryptedAccessToken:   linked.EncryptedAccessToken,
			TransactionCursor:      syncResult.NextTransactionCursor,
			ExternalUserID:         linked.ExternalUserID,
			Status:                 linked.Status,
			Products:               linked.Products,
		}); err != nil {
			itemResult.Status = "error"
			itemResult.ErrorCode = "CURSOR_UPDATE_FAILED"
			itemResult.ErrorMessage = err.Error()
			recordSyncRun(ctx, target, itemResult, started)
			return itemResult
		}
	}
	recordSyncRun(ctx, target, itemResult, started)
	return itemResult
}

func recordSyncRun(ctx context.Context, target Store, itemResult ItemResult, started string) {
	_ = target.RecordSyncRun(ctx, providers.SyncRun{
		Provider:             itemResult.Provider,
		ProviderItemID:       itemResult.ProviderItemID,
		StartedAt:            started,
		FinishedAt:           time.Now().UTC().Format(time.RFC3339),
		Status:               itemResult.Status,
		AccountsSeen:         itemResult.AccountsSeen,
		TransactionsAdded:    itemResult.TransactionsAdded,
		TransactionsModified: itemResult.TransactionsModified,
		TransactionsRemoved:  itemResult.TransactionsRemoved,
		RecurringStreamsSeen: itemResult.RecurringStreamsSeen,
		ErrorCode:            itemResult.ErrorCode,
		ErrorMessage:         itemResult.ErrorMessage,
	})
}
