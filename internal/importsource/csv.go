package importsource

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/thedavidweng/money/internal/core"
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

	headers := normalizeHeaders(records[0])
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

	var result Result
	accountsByName := make(map[string]core.Account)

	for i, row := range records[1:] {
		if len(row) == 0 {
			continue
		}

		date := getCell(row, colIndex, "date")
		amountStr := getCell(row, colIndex, "amount")
		name := getCell(row, colIndex, "name")
		currency := getCell(row, colIndex, "currency")
		accountName := getCell(row, colIndex, "account_name")
		if accountName == "" {
			accountName = "CSV Import"
		}

		amountMinor, err := parseCSVAmount(amountStr)
		if err != nil {
			result.DuplicatesSkipped++ // count parse failures as skipped
			continue
		}

		acc, ok := accountsByName[accountName]
		if !ok {
			accID, err := core.NewLocalID("acc_")
			if err != nil {
				return result, err
			}
			acc = core.Account{
				ID:                       accID,
				Name:                     accountName,
				Type:                     "other_asset",
				CurrentBalanceMinorUnits: 0,
				Currency:                 currency,
				Source:                   core.Source{Kind: "import", ImportSourceID: strPtr("csv"), ImportBatchID: strPtr(batchID)},
			}
			if err := store.UpsertImportedAccount(ctx, acc); err != nil {
				return result, fmt.Errorf("upsert account %q: %w", accountName, err)
			}
			accountsByName[accountName] = acc
			result.AccountsImported++
		}

		txID, err := core.NewLocalID("tx_")
		if err != nil {
			return result, err
		}

		rowHash := hashCSVRow(row)
		tx := core.Transaction{
			ID:               txID,
			AccountID:        acc.ID,
			Date:             date,
			AmountMinorUnits: amountMinor,
			Currency:         currency,
			Name:             name,
			CategorySource:   "none",
			Source:           core.Source{Kind: "import", ImportSourceID: strPtr("csv"), ImportBatchID: strPtr(batchID)},
		}

		inserted, possibleDups, err := store.UpsertImportedTransaction(ctx, tx, rowHash)
		if err != nil {
			return result, fmt.Errorf("upsert transaction row %d: %w", i+2, err)
		}
		if !inserted {
			result.DuplicatesSkipped++
			continue
		}
		result.TransactionsImported++
		if len(possibleDups) > 0 {
			result.PossibleDuplicates = append(result.PossibleDuplicates, txID)
		}
	}

	return result, nil
}

func normalizeHeaders(headers []string) []string {
	out := make([]string, len(headers))
	for i, h := range headers {
		h = strings.TrimSpace(strings.ToLower(h))
		h = strings.ReplaceAll(h, " ", "_")
		out[i] = h
	}
	return out
}

func getCell(row []string, colIndex map[string]int, col string) string {
	idx, ok := colIndex[col]
	if !ok || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func parseCSVAmount(amount string) (int64, error) {
	amount = strings.TrimSpace(amount)
	if amount == "" {
		return 0, fmt.Errorf("amount is empty")
	}
	// Remove currency symbols and thousands separators
	amount = strings.ReplaceAll(amount, ",", "")
	amount = strings.ReplaceAll(amount, "$", "")
	amount = strings.ReplaceAll(amount, "£", "")
	amount = strings.ReplaceAll(amount, "€", "")

	// Parse as float then convert to minor units
	f, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount %q", amount)
	}
	return int64(f * 100), nil
}

func hashCSVRow(row []string) string {
	h := sha256.New()
	for _, cell := range row {
		fmt.Fprintf(h, "%s\n", cell)
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

func strPtr(s string) *string {
	return &s
}
