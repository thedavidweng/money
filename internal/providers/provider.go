package providers

import (
	"context"

	"github.com/thedavidweng/money/internal/core"
)

type Provider interface {
	Name() string
	ValidateConfig(ctx context.Context) []ConfigDiagnostic
	SearchInstitutions(ctx context.Context, query string) ([]Institution, error)
	CreateLinkSession(ctx context.Context, request LinkRequest) (LinkSession, error)
	ExchangeLinkToken(ctx context.Context, session LinkSession, callback LinkCallback) (LinkedItem, error)
	Sync(ctx context.Context, item ProviderItem, sink SyncSink) (SyncResult, error)
}

// TransactionQuerier is an optional interface for providers that support
// querying transactions by date range without syncing.
type TransactionQuerier interface {
	QueryTransactions(ctx context.Context, item ProviderItem, startDate string, endDate string) ([]Transaction, error)
}

// HoldingQuerier is an optional interface for providers that support
// querying investment holdings.
type HoldingQuerier interface {
	QueryHoldings(ctx context.Context, item ProviderItem) (InvestmentHoldings, error)
}

// LiabilityQuerier is an optional interface for providers that support
// querying liabilities.
type LiabilityQuerier interface {
	QueryLiabilities(ctx context.Context, item ProviderItem) (Liabilities, error)
}

type InvestmentHoldings struct {
	Accounts   []FinancialAccount     `json:"accounts"`
	Holdings   []InvestmentHolding    `json:"holdings"`
	Securities []InvestmentSecurity   `json:"securities"`
}

type InvestmentHolding struct {
	AccountID        string   `json:"account_id"`
	SecurityID       string   `json:"security_id"`
	Quantity         float64  `json:"quantity"`
	InstitutionPrice float64  `json:"institution_price"`
	InstitutionValue float64  `json:"institution_value"`
	CostBasis        *float64 `json:"cost_basis,omitempty"`
	Currency         string   `json:"currency"`
}

type InvestmentSecurity struct {
	SecurityID      string  `json:"security_id"`
	ISIN            *string `json:"isin,omitempty"`
	CUSIP           *string `json:"cusip,omitempty"`
	SEDOL           *string `json:"sedol,omitempty"`
	Name            string  `json:"name"`
	TickerSymbol    *string `json:"ticker_symbol,omitempty"`
	Type            string  `json:"type"`
	ClosePrice      float64 `json:"close_price"`
	ClosePriceAsOf  *string `json:"close_price_as_of,omitempty"`
	Currency        string  `json:"currency"`
}

type Liabilities struct {
	Accounts    []FinancialAccount `json:"accounts"`
	Liabilities []Liability        `json:"liabilities"`
}

type Liability struct {
	AccountID       string  `json:"account_id"`
	Type            string  `json:"type"`
	CurrentBalance  float64 `json:"current_balance"`
	OriginalBalance *float64 `json:"original_balance,omitempty"`
	Currency        string  `json:"currency"`
	Name            string  `json:"name"`
	LastPaymentDate *string `json:"last_payment_date,omitempty"`
	LastPaymentAmount *float64 `json:"last_payment_amount,omitempty"`
	NextPaymentDueDate *string `json:"next_payment_due_date,omitempty"`
	APR             *float64 `json:"apr,omitempty"`
}

type SyncSink interface {
	UpsertInstitution(ctx context.Context, institution Institution) error
	UpsertProviderItem(ctx context.Context, item ProviderItem) error
	UpsertAccount(ctx context.Context, account FinancialAccount) error
	UpsertTransaction(ctx context.Context, transaction Transaction) error
	UpsertRecurring(ctx context.Context, recurring Recurring) error
	MarkTransactionRemoved(ctx context.Context, providerItemID string, providerTransactionID string) error
	RecordSyncRun(ctx context.Context, run SyncRun) error
	UpsertSecurity(ctx context.Context, security InvestmentSecurity) error
	UpsertHolding(ctx context.Context, providerItemID string, holding InvestmentHolding) error
	ClearHoldings(ctx context.Context, providerItemID string) error
	UpsertLiability(ctx context.Context, providerItemID string, liability Liability) error
	ClearLiabilities(ctx context.Context, providerItemID string) error
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

type Transaction struct {
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
	Source                core.Source
}

type Recurring struct {
	ProviderItemID          string
	ProviderRecurringID     string
	ProviderAccountID       string
	MerchantName            string
	AverageAmountMinorUnits int64
	Currency                string
	Frequency               string
	NextDate                *string
}

type ConfigDiagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type LinkRequest struct {
	Institution  Institution
	Products     []string
	CountryCodes []string
	RedirectURI  string
	State        string
}

type LinkSession struct {
	Provider                string   `json:"provider"`
	URL                     string   `json:"url,omitempty"`
	LinkToken               string   `json:"link_token,omitempty"`
	State                   string   `json:"state"`
	ProviderAccessToken     string   `json:"-"`
	ExistingProviderItemIDs []string `json:"-"`
}

type LinkCallback struct {
	PublicToken string
	State       string
	Metadata    LinkMetadata
}

type LinkMetadata struct {
	Institution LinkInstitutionMetadata `json:"institution"`
	Accounts    []LinkAccountMetadata   `json:"accounts"`
}

type LinkInstitutionMetadata struct {
	ID   string `json:"institution_id"`
	Name string `json:"name"`
}

type LinkAccountMetadata struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Mask    string `json:"mask"`
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
}

type LinkedItem struct {
	Institution  Institution
	ProviderItem ProviderItem
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
