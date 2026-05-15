package importsource

import (
	"context"
	"fmt"
	"io"

	"github.com/thedavidweng/money/internal/core"
)

// ImportSource transforms an external file into canonical local records.
type ImportSource interface {
	Name() string
	Import(ctx context.Context, store ImportStore, batchID string, r io.Reader) (Result, error)
}

// ImportStore is the minimal store surface needed by import sources.
type ImportStore interface {
	UpsertImportedAccount(ctx context.Context, account core.Account) error
	// UpsertImportedTransaction inserts or skips a transaction. Returns (inserted, possibleDuplicateIDs, error).
	UpsertImportedTransaction(ctx context.Context, tx core.Transaction, sourceRowHash string) (bool, []string, error)
}

// Result reports what an import source produced.
type Result struct {
	AccountsImported     int      `json:"accounts_imported"`
	TransactionsImported int      `json:"transactions_imported"`
	DuplicatesSkipped    int      `json:"duplicates_skipped"`
	PossibleDuplicates   []string `json:"possible_duplicates,omitempty"`
}

// Registry holds registered import sources by name.
type Registry struct {
	sources map[string]ImportSource
}

func NewRegistry() *Registry {
	return &Registry{sources: map[string]ImportSource{}}
}

func (r *Registry) Register(source ImportSource) {
	r.sources[source.Name()] = source
}

func (r *Registry) Get(name string) (ImportSource, bool) {
	s, ok := r.sources[name]
	return s, ok
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.sources))
	for name := range r.sources {
		names = append(names, name)
	}
	return names
}

// DefaultRegistry returns a registry with built-in import sources.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(&MonarchImporter{})
	r.Register(&CSVImporter{})
	return r
}

// ImportError is a stable error type for import failures.
type ImportError struct {
	Code    string
	Message string
}

func (e ImportError) Error() string {
	return fmt.Sprintf("import error %s: %s", e.Code, e.Message)
}
