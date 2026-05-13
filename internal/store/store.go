package store

import (
	"context"

	"github.com/thedavidweng/money/internal/core"
)

type Store interface {
	ListAccounts(ctx context.Context) ([]core.Account, error)
	CreateManualAccount(ctx context.Context, account core.Account) (core.Account, error)
	ListTransactions(ctx context.Context, query TransactionListQuery) ([]core.Transaction, error)
	SearchTransactions(ctx context.Context, query string, limit int) ([]core.Transaction, error)
	ListCategories(ctx context.Context) ([]core.Category, error)
	ListTags(ctx context.Context) ([]core.Tag, error)
	ListRecurring(ctx context.Context) ([]core.Recurring, error)
	ListProviderItems(ctx context.Context, query ProviderItemQuery) ([]LinkedItem, error)
	GetProviderItem(ctx context.Context, id string) (LinkedItem, error)
	UpdateProviderItemName(ctx context.Context, id string, name string) error
	RemoveProviderItem(ctx context.Context, id string) error
	ListHoldings(ctx context.Context) ([]core.InvestmentHolding, error)
	ListSecurities(ctx context.Context) ([]core.InvestmentSecurity, error)
	ListLiabilities(ctx context.Context) ([]core.Liability, error)
}

type TransactionListQuery struct {
	AccountID   string
	CategoryID  string
	Merchant    string
	TagID       string
	DateFrom    string
	DateTo      string
	NeedsReview *bool
	Pending     *bool
	Recurring   *bool
	RemovedMode RemovedMode
	Limit       int
	Offset      int
}

type RemovedMode string

const (
	RemovedExclude RemovedMode = "exclude"
	RemovedInclude RemovedMode = "include"
	RemovedOnly    RemovedMode = "only"
)
