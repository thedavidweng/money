package importsource

import (
	"bytes"
	"context"
	"testing"

	"github.com/thedavidweng/money/internal/core"
)

// noopStore satisfies ImportStore without any real persistence,
// so we can fuzz the CSV parsing pipeline for panics.
type noopStore struct{}

func (noopStore) UpsertImportedAccount(_ context.Context, _ *core.Account) error {
	return nil
}

func (noopStore) UpsertImportedTransaction(_ context.Context, _ *core.Transaction, _ string) (inserted bool, duplicateIDs []string, err error) {
	return true, nil, nil
}

func FuzzParseCSV(f *testing.F) {
	// Seed corpus with valid and edge-case CSV inputs.
	f.Add([]byte("date,amount,name,currency\n2024-01-01,-10.50,Grocery,USD\n"))
	f.Add([]byte("date,amount,name,currency\n"))
	f.Add([]byte(""))
	f.Add([]byte(",,,\n,,,\n"))
	f.Add([]byte("\"date\",\"amount\",\"name\",\"currency\"\n\"2024-06-01\",\"$1,234.56\",\"Coffee Shop\",\"USD\"\n"))
	f.Add([]byte("date,amount,name,currency,account_name\n2024-01-01,-10,Test,USD,Checking\n"))

	var imp CSVImporter
	ctx := context.Background()
	store := noopStore{}

	f.Fuzz(func(t *testing.T, data []byte) {
		// Must not panic on any byte sequence.
		_, _ = imp.Import(ctx, store, "fuzz-batch", bytes.NewReader(data))
	})
}
