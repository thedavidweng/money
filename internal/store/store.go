// Package store defines the persistence interface and provides an encrypted
// SQLite implementation for accounts, transactions, budgets, rules, and
// provider-linked data.
package store

import (
	"context"

	"github.com/thedavidweng/money/internal/core"
)

type Store interface {
	// Read-side
	ListAccounts(ctx context.Context) ([]core.Account, error)
	CreateManualAccount(ctx context.Context, account *core.Account) (core.Account, error)
	ListTransactions(ctx context.Context, query *TransactionListQuery) ([]core.Transaction, error)
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
	UpsertImportedAccount(ctx context.Context, account *core.Account) error
	UpsertImportedTransaction(ctx context.Context, tx *core.Transaction, sourceRowHash string) (bool, []string, error)
	CashflowSummary(ctx context.Context, from, to, period, currency string) ([]core.CashflowPeriod, error)
	NetWorth(ctx context.Context) (core.NetWorth, error)
	ListBudgets(ctx context.Context) ([]core.Budget, error)
	CreateBudget(ctx context.Context, budget *core.Budget) (core.Budget, error)
	UpdateBudget(ctx context.Context, budget *core.Budget) (core.Budget, error)
	DeleteBudget(ctx context.Context, id string) error
	GetBudget(ctx context.Context, id string) (core.Budget, error)
	ListBudgetCategories(ctx context.Context, budgetID string) ([]core.BudgetCategory, error)
	CreateBudgetCategory(ctx context.Context, bc *core.BudgetCategory) (core.BudgetCategory, error)
	DeleteBudgetCategory(ctx context.Context, id string) error
	ListRules(ctx context.Context) ([]core.Rule, error)
	CreateRule(ctx context.Context, rule *core.Rule) (core.Rule, error)
	UpdateRule(ctx context.Context, rule *core.Rule) (core.Rule, error)
	DeleteRule(ctx context.Context, id string) error
	ApplyRules(ctx context.Context) (core.ApplyRulesResult, error)

	// Write-side (sync + link)
	UpsertInstitution(ctx context.Context, institution core.Institution) error
	UpsertProviderItem(ctx context.Context, item *core.ProviderItem) error
	UpsertAccount(ctx context.Context, account *core.FinancialAccount) error
	UpsertTransaction(ctx context.Context, transaction *core.ProviderTransaction) error
	UpsertRecurring(ctx context.Context, recurring *core.ProviderRecurring) error
	MarkTransactionRemoved(ctx context.Context, providerItemID string, providerTransactionID string) error
	RecordSyncRun(ctx context.Context, run *core.SyncRun) error
	UpsertSecurity(ctx context.Context, security *core.InvestmentSecurity) error
	UpsertHolding(ctx context.Context, providerItemID string, holding *core.InvestmentHolding) error
	ClearHoldings(ctx context.Context, providerItemID string) error
	UpsertLiability(ctx context.Context, providerItemID string, liability *core.Liability) error
	ClearLiabilities(ctx context.Context, providerItemID string) error
	StoreLinkedProviderItem(ctx context.Context, linked *LinkedProviderItem) error
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
