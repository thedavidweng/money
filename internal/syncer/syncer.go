package syncer

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/thedavidweng/money/internal/providers"
	"github.com/thedavidweng/money/internal/store"
)

type Registry interface {
	Get(name string) (providers.Provider, bool)
}

type Options struct {
	Provider       string
	ProviderItemID string
	StartDate      string
	EndDate        string
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

func Sync(ctx context.Context, target store.Store, registry Registry, options Options) (Result, error) {
	linkedItems, err := target.ListProviderItems(ctx, store.ProviderItemQuery{Provider: options.Provider, ProviderItemID: options.ProviderItemID})
	if err != nil {
		return Result{}, err
	}
	if len(linkedItems) == 0 {
		return Result{Warnings: []Warning{{Code: "NO_LINKED_PROVIDER_ITEMS", Message: "No linked Provider Items found."}}}, nil
	}

	result := Result{Items: []ItemResult{}, Warnings: []Warning{}}
	var failed bool
	for i := range linkedItems {
		itemResult := syncOne(ctx, target, registry, &linkedItems[i], options.StartDate, options.EndDate)
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

func syncOne(ctx context.Context, target store.Store, registry Registry, linked *store.LinkedItem, startDate, endDate string) ItemResult {
	started := time.Now().UTC().Format(time.RFC3339)
	itemResult := ItemResult{Provider: linked.Provider, ProviderItemID: linked.ID, Status: "ok"}
	provider, ok := registry.Get(linked.Provider)
	if !ok {
		itemResult.Status = "error"
		itemResult.ErrorCode = "PROVIDER_NOT_REGISTERED"
		itemResult.ErrorMessage = fmt.Sprintf("provider %q is not registered", linked.Provider)
		recordSyncRun(ctx, target, &itemResult, started)
		return itemResult
	}
	syncResult, err := provider.Sync(ctx, &providers.ProviderItem{
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
		recordSyncRun(ctx, target, &itemResult, started)
		return itemResult
	}
	itemResult.AccountsSeen = syncResult.AccountsSeen
	itemResult.TransactionsAdded = syncResult.TransactionsAdded
	itemResult.TransactionsModified = syncResult.TransactionsModified
	itemResult.TransactionsRemoved = syncResult.TransactionsRemoved
	itemResult.RecurringStreamsSeen = syncResult.RecurringStreamsSeen
	itemResult.NextTransactionCursor = syncResult.NextTransactionCursor

	// If a date range is specified and the provider supports TransactionQuerier, backfill transactions for that range.
	if startDate != "" && endDate != "" {
		if querier, ok := provider.(providers.TransactionQuerier); ok {
			backfillTxs, err := querier.QueryTransactions(ctx, &providers.ProviderItem{
				ID:                     linked.ID,
				Provider:               linked.Provider,
				InstitutionID:          linked.InstitutionID,
				ProviderExternalItemID: linked.ProviderExternalItemID,
				EncryptedAccessToken:   linked.EncryptedAccessToken,
				TransactionCursor:      linked.TransactionCursor,
				ExternalUserID:         linked.ExternalUserID,
				Status:                 linked.Status,
				Products:               linked.Products,
			}, startDate, endDate)
			if err != nil {
				classified := providers.ClassifyProviderError(linked.Provider, err)
				itemResult.Status = "error"
				itemResult.ErrorCode = classified.Code
				itemResult.ErrorMessage = "date-range backfill failed: " + classified.Message
				recordSyncRun(ctx, target, &itemResult, started)
				return itemResult
			}
			for i := range backfillTxs {
				if err := target.UpsertTransaction(ctx, &backfillTxs[i]); err != nil {
					itemResult.Status = "error"
					itemResult.ErrorCode = "BACKFILL_TRANSACTION_UPSERT_FAILED"
					itemResult.ErrorMessage = err.Error()
					recordSyncRun(ctx, target, &itemResult, started)
					return itemResult
				}
				itemResult.TransactionsAdded++
			}
		}
	}

	// Sync investment holdings if provider supports them and the item has the product.
	if holder, ok := provider.(providers.HoldingQuerier); ok && hasProduct(linked.Products, "investments") {
		holdings, err := holder.QueryHoldings(ctx, providers.ProviderItem{
			ID:                     linked.ID,
			Provider:               linked.Provider,
			InstitutionID:          linked.InstitutionID,
			ProviderExternalItemID: linked.ProviderExternalItemID,
			EncryptedAccessToken:   linked.EncryptedAccessToken,
			TransactionCursor:      linked.TransactionCursor,
			ExternalUserID:         linked.ExternalUserID,
			Status:                 linked.Status,
			Products:               linked.Products,
		})
		if err != nil {
			classified := providers.ClassifyProviderError(linked.Provider, err)
			itemResult.Status = "error"
			itemResult.ErrorCode = classified.Code
			itemResult.ErrorMessage = "holdings sync failed: " + classified.Message
			recordSyncRun(ctx, target, &itemResult, started)
			return itemResult
		}
		if err := target.ClearHoldings(ctx, linked.ID); err != nil {
			itemResult.Status = "error"
			itemResult.ErrorCode = "CLEAR_HOLDINGS_FAILED"
			itemResult.ErrorMessage = err.Error()
			recordSyncRun(ctx, target, &itemResult, started)
			return itemResult
		}
		for i := range holdings.Securities {
			err := target.UpsertSecurity(ctx, &holdings.Securities[i])
			if err == nil {
				continue
			}
			itemResult.Status = "error"
			itemResult.ErrorCode = "SECURITY_UPSERT_FAILED"
			itemResult.ErrorMessage = err.Error()
			recordSyncRun(ctx, target, &itemResult, started)
			return itemResult
		}
		for i := range holdings.Holdings {
			err := target.UpsertHolding(ctx, linked.ID, &holdings.Holdings[i])
			if err == nil {
				continue
			}
			itemResult.Status = "error"
			itemResult.ErrorCode = "HOLDING_UPSERT_FAILED"
			itemResult.ErrorMessage = err.Error()
			recordSyncRun(ctx, target, &itemResult, started)
			return itemResult
		}
	}

	// Sync liabilities if provider supports them and the item has the product.
	if liabilityQuerier, ok := provider.(providers.LiabilityQuerier); ok && hasProduct(linked.Products, "liabilities") {
		liabilities, err := liabilityQuerier.QueryLiabilities(ctx, providers.ProviderItem{
			ID:                     linked.ID,
			Provider:               linked.Provider,
			InstitutionID:          linked.InstitutionID,
			ProviderExternalItemID: linked.ProviderExternalItemID,
			EncryptedAccessToken:   linked.EncryptedAccessToken,
			TransactionCursor:      linked.TransactionCursor,
			ExternalUserID:         linked.ExternalUserID,
			Status:                 linked.Status,
			Products:               linked.Products,
		})
		if err != nil {
			classified := providers.ClassifyProviderError(linked.Provider, err)
			itemResult.Status = "error"
			itemResult.ErrorCode = classified.Code
			itemResult.ErrorMessage = "liabilities sync failed: " + classified.Message
			recordSyncRun(ctx, target, &itemResult, started)
			return itemResult
		}
		if err := target.ClearLiabilities(ctx, linked.ID); err != nil {
			itemResult.Status = "error"
			itemResult.ErrorCode = "CLEAR_LIABILITIES_FAILED"
			itemResult.ErrorMessage = err.Error()
			recordSyncRun(ctx, target, &itemResult, started)
			return itemResult
		}
		for i := range liabilities.Liabilities {
			err := target.UpsertLiability(ctx, linked.ID, &liabilities.Liabilities[i])
			if err == nil {
				continue
			}
			itemResult.Status = "error"
			itemResult.ErrorCode = "LIABILITY_UPSERT_FAILED"
			itemResult.ErrorMessage = err.Error()
			recordSyncRun(ctx, target, &itemResult, started)
			return itemResult
		}
	}

	if syncResult.NextTransactionCursor != "" {
		if err := target.UpsertProviderItem(ctx, &providers.ProviderItem{
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
			recordSyncRun(ctx, target, &itemResult, started)
			return itemResult
		}
	}
	recordSyncRun(ctx, target, &itemResult, started)
	return itemResult
}

func hasProduct(products []string, target string) bool {
	for _, p := range products {
		if p == target {
			return true
		}
	}
	return false
}

func recordSyncRun(ctx context.Context, target store.Store, itemResult *ItemResult, started string) {
	if err := target.RecordSyncRun(ctx, &providers.SyncRun{
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
	}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to record sync run: %v\n", err)
	}
}
