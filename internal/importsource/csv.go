package importsource

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
)

// CSVImporter reads CSV files and maps them to canonical records.
// Expected columns: date, amount, name, currency, account_name (optional).
// Amount should use negative for outflow, positive for inflow.
type CSVImporter struct{}

func (c *CSVImporter) Name() string {
	return "csv"
}

func (c *CSVImporter) Import(ctx context.Context, store ImportStore, batchID string, r io.Reader) (Result, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return Result{}, ImportError{Code: "INVALID_CSV", Message: err.Error()}
	}
	if len(records) < 2 {
		return Result{}, ImportError{Code: "EMPTY_CSV", Message: "CSV has no data rows"}
	}

	headers := records[0]
	colIndex := make(map[string]int)
	for i, h := range headers {
		colIndex[h] = i
	}

	required := []string{"date", "amount", "name", "currency"}
	for _, col := range required {
		if _, ok := colIndex[col]; !ok {
			return Result{}, ImportError{Code: "MISSING_COLUMN", Message: fmt.Sprintf("required column %q not found", col)}
		}
	}

	// CSV import is deferred to a follow-up commit for full implementation.
	return Result{}, ImportError{Code: "NOT_IMPLEMENTED", Message: "CSV import is not fully implemented yet"}
}
