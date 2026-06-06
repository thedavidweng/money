package importsource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/thedavidweng/money/internal/core"
)

// MonarchImporter reads Monarch JSON exports and maps them to canonical records.
// Expected JSON shape:
//   {
//     "accounts": [...],
//     "transactions": [...]
//   }
// Each account has: id, name, type, subtype, balance, currency.
// Each transaction has: id, account_id, date, amount, name, merchant_name, category, pending.
type MonarchImporter struct{}

func (m *MonarchImporter) Name() string {
	return "monarch"
}

func (m *MonarchImporter) Import(ctx context.Context, store ImportStore, batchID string, r io.Reader) (Result, error) {
	var payload struct {
		Accounts     []monarchAccount     `json:"accounts"`
		Transactions []monarchTransaction `json:"transactions"`
	}
	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return Result{}, ImportError{Code: "INVALID_JSON", Message: err.Error()}
	}

	result := Result{}
	accountIDMap := make(map[string]string) // monarch account ID -> local account ID
	now := time.Now().UTC().Format(time.RFC3339)

	for _, ma := range payload.Accounts {
		localID, err := core.NewLocalID("acc_")
		if err != nil {
			return result, ImportError{Code: "ID_GENERATION_FAILED", Message: err.Error()}
		}
		accountIDMap[ma.ID] = localID

		acc := core.Account{
			ID:                       localID,
			Name:                     ma.Name,
			Type:                     mapMonarchType(ma.Type),
			Subtype:                  ma.Subtype,
			CurrentBalanceMinorUnits: int64(ma.Balance * 100),
			Currency:                 ma.Currency,
			Source: core.Source{
				Kind:           "import",
				ImportSourceID: stringPtr("monarch"),
				ImportBatchID:  stringPtr(batchID),
			},
			UpdatedAt: now,
		}
		if err := store.UpsertImportedAccount(ctx, acc); err != nil {
			return result, ImportError{Code: "ACCOUNT_WRITE_FAILED", Message: err.Error()}
		}
		result.AccountsImported++
	}

	for _, mt := range payload.Transactions {
		accountID, ok := accountIDMap[mt.AccountID]
		if !ok {
			return result, ImportError{Code: "UNKNOWN_ACCOUNT", Message: fmt.Sprintf("transaction references unknown account %q", mt.AccountID)}
		}

		rowHash := hashMonarchRow(mt)
		txID, err := core.NewLocalID("tx_")
		if err != nil {
			return result, ImportError{Code: "ID_GENERATION_FAILED", Message: err.Error()}
		}
		tx := core.Transaction{
			ID:               txID,
			AccountID:        accountID,
			Date:             mt.Date,
			AmountMinorUnits: int64(mt.Amount * 100),
			Currency:         mt.Currency,
			Name:             mt.Name,
			MerchantName:     mt.MerchantName,
			Pending:          mt.Pending,
			CategorySource:   "import",
			Source: core.Source{
				Kind:           "import",
				ImportSourceID: stringPtr("monarch"),
				ImportBatchID:  stringPtr(batchID),
			},
			LastChangedAt: now,
		}
		if mt.Category != "" {
			tx.CategoryName = &mt.Category
		}

		inserted, possibleDups, err := store.UpsertImportedTransaction(ctx, tx, rowHash)
		if err != nil {
			return result, ImportError{Code: "TRANSACTION_WRITE_FAILED", Message: err.Error()}
		}
		if inserted {
			result.TransactionsImported++
		} else {
			result.DuplicatesSkipped++
		}
		result.PossibleDuplicates = append(result.PossibleDuplicates, possibleDups...)
	}

	return result, nil
}

type monarchAccount struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Subtype  string  `json:"subtype"`
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}

type monarchTransaction struct {
	ID           string  `json:"id"`
	AccountID    string  `json:"account_id"`
	Date         string  `json:"date"`
	Amount       float64 `json:"amount"`
	Currency     string  `json:"currency"`
	Name         string  `json:"name"`
	MerchantName string  `json:"merchant_name"`
	Category     string  `json:"category"`
	Pending      bool    `json:"pending"`
}

func mapMonarchType(monarchType string) string {
	switch monarchType {
	case "CHECKING", "SAVINGS":
		return "depository"
	case "CREDIT_CARD":
		return "credit"
	case "INVESTMENT":
		return "investment"
	case "LOAN", "MORTGAGE":
		return "loan"
	case "PROPERTY":
		return "property"
	case "VEHICLE":
		return "vehicle"
	default:
		return "other_asset"
	}
}

func hashMonarchRow(mt monarchTransaction) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%s|%s|%d|%s|%s|%t", mt.ID, mt.AccountID, mt.Date, int64(mt.Amount*100), mt.Currency, mt.Name, mt.Pending)
	return hex.EncodeToString(h.Sum(nil))
}

func stringPtr(s string) *string {
	return &s
}
