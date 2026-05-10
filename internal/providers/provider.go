package providers

import "context"

type Provider interface {
	Name() string
	Sync(ctx context.Context, sink SyncSink) (SyncResult, error)
}

type SyncSink interface {
	UpsertInstitution(ctx context.Context, institution Institution) error
	UpsertAccount(ctx context.Context, account Account) error
	UpsertTransaction(ctx context.Context, transaction Transaction) error
}

type SyncResult struct {
	Provider          string `json:"provider"`
	InstitutionsSeen  int    `json:"institutions_seen"`
	AccountsSeen      int    `json:"accounts_seen"`
	TransactionsAdded int    `json:"transactions_added"`
	TransactionsSeen  int    `json:"transactions_seen"`
}

type Institution struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}

type Account struct {
	ID            string  `json:"id"`
	InstitutionID string  `json:"institution_id"`
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Subtype       string  `json:"subtype"`
	Balance       float64 `json:"balance"`
	Currency      string  `json:"currency"`
}

type Transaction struct {
	ID           string  `json:"id"`
	AccountID    string  `json:"account_id"`
	Date         string  `json:"date"`
	Amount       float64 `json:"amount"`
	Name         string  `json:"name"`
	MerchantName string  `json:"merchant_name"`
	Category     string  `json:"category"`
	Pending      bool    `json:"pending"`
	Currency     string  `json:"currency"`
}
