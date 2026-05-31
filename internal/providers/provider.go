package providers

import (
	"context"

	"github.com/thedavidweng/money/internal/core"
)

// Type aliases: these types are defined in core and re-exported here
// so that provider adapters can use them without importing core directly.
// The store package imports core directly — it no longer imports providers.
type (
	Institution          = core.Institution
	ProviderItem         = core.ProviderItem
	FinancialAccount     = core.FinancialAccount
	Transaction          = core.ProviderTransaction
	Recurring            = core.ProviderRecurring
	SyncRun              = core.SyncRun
	SyncResult           = core.SyncResult
	InvestmentHolding    = core.InvestmentHolding
	InvestmentSecurity   = core.InvestmentSecurity
	Liability            = core.Liability
	SyncSink             = core.SyncSink
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
	Accounts   []FinancialAccount   `json:"accounts"`
	Holdings   []InvestmentHolding  `json:"holdings"`
	Securities []InvestmentSecurity `json:"securities"`
}

type Liabilities struct {
	Accounts    []FinancialAccount `json:"accounts"`
	Liabilities []Liability        `json:"liabilities"`
}

type ConfigDiagnostic struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

type LinkRequest struct {
	Institution                 Institution
	Products                    []string
	CountryCodes                []string
	RedirectURI                 string
	State                       string
	AdditionalConsentedProducts []string
	RequiredIfSupportedProducts []string
	OptionalProducts            []string
}

type SandboxPublicTokenRequest struct {
	InstitutionID string
	Products      []string
}

type SandboxPublicTokenCreator interface {
	CreateSandboxPublicToken(ctx context.Context, request SandboxPublicTokenRequest) (string, error)
}

type LinkSession struct {
	Provider                string   `json:"provider"`
	URL                     string   `json:"url,omitempty"`
	LinkToken               string   `json:"link_token,omitempty"`
	State                   string   `json:"state"`
	Products                []string `json:"products,omitempty"`
	ProviderAccessToken     string   `json:"-"`
	ExistingProviderItemIDs []string `json:"-"`
}

type LinkCallback struct {
	PublicToken string
	State       string
	Status      string
	Metadata    LinkMetadata
	Error       LinkError
}

type LinkMetadata struct {
	Institution   LinkInstitutionMetadata `json:"institution"`
	Accounts      []LinkAccountMetadata   `json:"accounts"`
	RequestID     string                  `json:"request_id"`
	LinkSessionID string                  `json:"link_session_id"`
}

type LinkError struct {
	Type           string `json:"error_type"`
	Code           string `json:"error_code"`
	Message        string `json:"error_message"`
	DisplayMessage string `json:"display_message"`
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
