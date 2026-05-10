package core

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strconv"
	"strings"
)

type Money struct {
	MinorUnits int64  `json:"-"`
	Currency   string `json:"currency"`
}

func FormatMinorUnits(minorUnits int64, currency string) string {
	sign := ""
	if minorUnits < 0 {
		sign = "-"
		minorUnits = -minorUnits
	}
	return fmt.Sprintf("%s%d.%02d", sign, minorUnits/100, minorUnits%100)
}

func ParseUnsignedDecimalMinorUnits(input string) (int64, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(input), ",", "")
	if normalized == "" {
		return 0, fmt.Errorf("amount is required")
	}
	if strings.HasPrefix(normalized, "+") || strings.HasPrefix(normalized, "-") {
		return 0, fmt.Errorf("amount must be unsigned")
	}

	whole, frac, hasFrac := strings.Cut(normalized, ".")
	if whole == "" {
		whole = "0"
	}
	if !onlyDigits(whole) {
		return 0, fmt.Errorf("amount must contain only digits, thousands separators, and one decimal point")
	}
	if !hasFrac {
		frac = "00"
	}
	if len(frac) > 2 || !onlyDigits(frac) {
		return 0, fmt.Errorf("amount must use at most two decimal places")
	}
	for len(frac) < 2 {
		frac += "0"
	}
	wholeUnits, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount is too large")
	}
	fractionUnits, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("amount is invalid")
	}
	return wholeUnits*100 + fractionUnits, nil
}

func SignedManualBalance(accountType string, unsignedMinorUnits int64) (int64, string, error) {
	switch accountType {
	case "credit", "loan", "other_liability":
		return -unsignedMinorUnits, "decreases", nil
	case "depository", "investment", "property", "vehicle", "other_asset":
		return unsignedMinorUnits, "increases", nil
	default:
		return 0, "", fmt.Errorf("unsupported manual account type %q", accountType)
	}
}

func NewLocalID(prefix string) (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	suffix := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw[:]))
	return prefix + suffix, nil
}

func onlyDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

type Source struct {
	Kind                   string  `json:"kind"`
	Provider               *string `json:"provider"`
	ProviderItemID         *string `json:"provider_item_id"`
	ProviderExternalItemID *string `json:"provider_external_item_id"`
	InstitutionID          *string `json:"institution_id"`
	ProviderAccountID      *string `json:"provider_account_id"`
	ProviderTransactionID  *string `json:"provider_transaction_id"`
	ImportSourceID         *string `json:"import_source_id"`
	ImportBatchID          *string `json:"import_batch_id"`
}

type Account struct {
	ID                         string  `json:"id"`
	DisplayName                string  `json:"display_name"`
	Name                       string  `json:"name,omitempty"`
	OfficialName               string  `json:"official_name,omitempty"`
	Alias                      string  `json:"alias,omitempty"`
	Mask                       string  `json:"mask,omitempty"`
	InstitutionID              string  `json:"institution_id,omitempty"`
	Type                       string  `json:"type"`
	Subtype                    string  `json:"subtype,omitempty"`
	CurrentBalanceMinorUnits   int64   `json:"-"`
	CurrentBalance             string  `json:"current_balance"`
	AvailableBalanceMinorUnits *int64  `json:"-"`
	AvailableBalance           *string `json:"available_balance,omitempty"`
	AvailableCreditMinorUnits  *int64  `json:"-"`
	AvailableCredit            *string `json:"available_credit,omitempty"`
	Currency                   string  `json:"currency"`
	Source                     Source  `json:"source"`
	Hidden                     bool    `json:"hidden,omitempty"`
	UpdatedAt                  string  `json:"updated_at,omitempty"`
}

type Transaction struct {
	ID                     string   `json:"id"`
	AccountID              string   `json:"account_id"`
	Date                   string   `json:"date"`
	AuthorizedDate         *string  `json:"authorized_date,omitempty"`
	Datetime               *string  `json:"datetime,omitempty"`
	AuthorizedDatetime     *string  `json:"authorized_datetime,omitempty"`
	AmountMinorUnits       int64    `json:"-"`
	Amount                 string   `json:"amount"`
	Name                   string   `json:"name"`
	MerchantName           string   `json:"merchant_name,omitempty"`
	CategoryID             *string  `json:"category_id"`
	CategoryName           *string  `json:"category_name"`
	CategorySource         string   `json:"category_source"`
	ProviderCategory       *string  `json:"provider_category,omitempty"`
	ProviderSubcategory    *string  `json:"provider_subcategory,omitempty"`
	Pending                bool     `json:"pending"`
	Removed                bool     `json:"removed,omitempty"`
	Currency               string   `json:"currency"`
	Source                 Source   `json:"source"`
	NeedsReview            bool     `json:"needs_review"`
	Note                   *string  `json:"note"`
	TagIDs                 []string `json:"tag_ids"`
	Tags                   []Tag    `json:"tags"`
	RecurringTransactionID *string  `json:"recurring_transaction_id,omitempty"`
	LastChangedAt          string   `json:"last_changed_at,omitempty"`
}

type Category struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	GroupName *string `json:"group_name,omitempty"`
	Hidden    bool    `json:"hidden,omitempty"`
	UpdatedAt string  `json:"updated_at,omitempty"`
}

type Tag struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

type Recurring struct {
	ID                      string  `json:"id"`
	AccountID               string  `json:"account_id"`
	MerchantName            string  `json:"merchant_name"`
	AverageAmount           string  `json:"average_amount"`
	AverageAmountMinorUnits int64   `json:"-"`
	Currency                string  `json:"currency"`
	Frequency               string  `json:"frequency"`
	NextDate                *string `json:"next_date"`
	Source                  Source  `json:"source"`
	UpdatedAt               string  `json:"updated_at,omitempty"`
}
