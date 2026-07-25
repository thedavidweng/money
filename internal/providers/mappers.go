package providers

import "math"

type PlaidTransaction struct {
	ProviderItemID        string
	ProviderTransactionID string
	ProviderAccountID     string
	Date                  string
	AuthorizedDate        *string
	Amount                float64
	Name                  string
	MerchantName          string
	Currency              string
	Pending               bool
	ProviderCategory      *string
	ProviderSubcategory   *string
}

type PlaidAccount struct {
	ProviderItemID    string
	ProviderAccountID string
	Name              string
	OfficialName      string
	Mask              string
	Type              string
	Subtype           string
	CurrentBalance    float64
	AvailableBalance  *float64
	AvailableCredit   *float64
	Currency          string
}

type BridgeTransaction struct {
	ProviderItemID        string
	ProviderTransactionID string
	ProviderAccountID     string
	Date                  string
	Amount                float64
	Direction             string
	Description           string
	MerchantName          string
	Currency              string
	Pending               bool
}

func MapPlaidTransaction(input *PlaidTransaction) Transaction {
	merchant := input.MerchantName
	if merchant == "" {
		merchant = input.Name
	}
	return Transaction{
		ProviderItemID:        input.ProviderItemID,
		ProviderTransactionID: input.ProviderTransactionID,
		ProviderAccountID:     input.ProviderAccountID,
		Date:                  input.Date,
		AuthorizedDate:        input.AuthorizedDate,
		AmountMinorUnits:      -minorUnits(input.Amount),
		Name:                  input.Name,
		MerchantName:          merchant,
		ProviderCategory:      input.ProviderCategory,
		ProviderSubcategory:   input.ProviderSubcategory,
		Pending:               input.Pending,
		Currency:              input.Currency,
	}
}

func MapPlaidAccount(input *PlaidAccount) FinancialAccount {
	currentBalance := minorUnits(input.CurrentBalance)
	if liabilityAccount(input.Type) {
		currentBalance = -currentBalance
	}
	return FinancialAccount{
		ProviderItemID:             input.ProviderItemID,
		ProviderAccountID:          input.ProviderAccountID,
		Name:                       input.Name,
		OfficialName:               input.OfficialName,
		Mask:                       input.Mask,
		Type:                       input.Type,
		Subtype:                    input.Subtype,
		CurrentBalanceMinorUnits:   currentBalance,
		AvailableBalanceMinorUnits: optionalMinorUnits(input.AvailableBalance),
		AvailableCreditMinorUnits:  optionalMinorUnits(input.AvailableCredit),
		Currency:                   input.Currency,
	}
}

func MapBridgeTransaction(input *BridgeTransaction) Transaction {
	amount := minorUnits(input.Amount)
	if input.Direction == "debit" {
		amount = -amount
	}
	merchant := input.MerchantName
	if merchant == "" {
		merchant = input.Description
	}
	return Transaction{
		ProviderItemID:        input.ProviderItemID,
		ProviderTransactionID: input.ProviderTransactionID,
		ProviderAccountID:     input.ProviderAccountID,
		Date:                  input.Date,
		AmountMinorUnits:      amount,
		Name:                  input.Description,
		MerchantName:          merchant,
		Pending:               input.Pending,
		Currency:              input.Currency,
	}
}

func minorUnits(value float64) int64 {
	return int64(math.Round(value * 100))
}

func optionalMinorUnits(value *float64) *int64 {
	if value == nil {
		return nil
	}
	minor := minorUnits(*value)
	return &minor
}

func liabilityAccount(accountType string) bool {
	return accountType == "credit" || accountType == "loan"
}
