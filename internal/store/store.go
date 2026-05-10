package store

import (
	"context"

	"github.com/thedavidweng/money/internal/core"
)

type Store interface {
	ListAccounts(ctx context.Context) ([]core.Account, error)
	SearchTransactions(ctx context.Context, query string, limit int) ([]core.Transaction, error)
}
