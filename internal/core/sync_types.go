package core

import "context"

// Sync-related types used by the store's write-side methods.
// These types represent raw data from providers before local enrichment.
// The store imports core instead of providers, inverting the dependency.

type Institution struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Provider              string `json:"provider"`
	ProviderInstitutionID string `json:"provider_institution_id"`
}

type ProviderItem struct {
	ID                     string   `json:"id"`
	Provider               string   `json:"provider"`
	InstitutionID          string   `json:"institution_id"`
	ProviderExternalItemID string   `json:"provider_external_item_id"`
	EncryptedAccessToken   []byte   `json:"-"`
	TransactionCursor      string   `json:"transaction_cursor,omitempty"`
	ExternalUserID         string   `json:"external_user_id,omitempty"`
	Status                 string   `json:"status"`
	Products               []string `json:"products"`
}

type FinancialAccount struct {
	ProviderItemID             string
	ProviderAccountID          string
	Name                       string
	OfficialName               string
	Mask                       string
	Type                       string
	Subtype                    string
	CurrentBalanceMinorUnits   int64
	AvailableBalanceMinorUnits *int64
	AvailableCreditMinorUnits  *int64
	Currency                   string
	UpdatedAt                  string
}

type ProviderTransaction struct {
	ProviderItemID        string
	ProviderTransactionID string
	ProviderAccountID     string
	Date                  string
	AuthorizedDate        *string
	AmountMinorUnits      int64
	Name                  string
	MerchantName          string
	ProviderCategory      *string
	ProviderSubcategory   *string
	Pending               bool
	Currency              string
	Source                Source
}

type ProviderRecurring struct {
	ProviderItemID          string
	ProviderRecurringID     string
	ProviderAccountID       string
	MerchantName            string
	AverageAmountMinorUnits int64
	Currency                string
	Frequency               string
	NextDate                *string
}

type SyncRun struct {
	Provider             string
	ProviderItemID       string
	StartedAt            string
	FinishedAt           string
	Status               string
	AccountsSeen         int
	TransactionsAdded    int
	TransactionsModified int
	TransactionsRemoved  int
	RecurringStreamsSeen int
	ErrorCode            string
	ErrorMessage         string
}

type SyncResult struct {
	Provider              string `json:"provider"`
	ProviderItemID        string `json:"provider_item_id"`
	InstitutionsSeen      int    `json:"institutions_seen"`
	AccountsSeen          int    `json:"accounts_seen"`
	TransactionsAdded     int    `json:"transactions_added"`
	TransactionsModified  int    `json:"transactions_modified"`
	TransactionsRemoved   int    `json:"transactions_removed"`
	RecurringStreamsSeen  int    `json:"recurring_streams_seen"`
	NextTransactionCursor string `json:"next_transaction_cursor,omitempty"`
}

// SyncSink is the write-side interface for storing synced provider data.
type SyncSink interface {
	UpsertInstitution(ctx context.Context, institution Institution) error
	UpsertProviderItem(ctx context.Context, item *ProviderItem) error
	UpsertAccount(ctx context.Context, account *FinancialAccount) error
	UpsertTransaction(ctx context.Context, transaction *ProviderTransaction) error
	UpsertRecurring(ctx context.Context, recurring *ProviderRecurring) error
	MarkTransactionRemoved(ctx context.Context, providerItemID string, providerTransactionID string) error
	RecordSyncRun(ctx context.Context, run *SyncRun) error
	UpsertSecurity(ctx context.Context, security *InvestmentSecurity) error
	UpsertHolding(ctx context.Context, providerItemID string, holding *InvestmentHolding) error
	ClearHoldings(ctx context.Context, providerItemID string) error
	UpsertLiability(ctx context.Context, providerItemID string, liability *Liability) error
	ClearLiabilities(ctx context.Context, providerItemID string) error
}
