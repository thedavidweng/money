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

type SyncSink interface {
	UpsertInstitution(ctx context.Context, institution Institution) error
	UpsertProviderItem(ctx context.Context, item ProviderItem) error
	UpsertAccount(ctx context.Context, account FinancialAccount) error
	UpsertTransaction(ctx context.Context, transaction Transaction) error
	UpsertRecurring(ctx context.Context, recurring Recurring) error
	MarkTransactionRemoved(ctx context.Context, providerItemID string, providerTransactionID string) error
	RecordSyncRun(ctx context.Context, run SyncRun) error
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
