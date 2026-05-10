package providers

import "testing"

func TestMapPlaidTransactionNormalizesOutflowToNegative(t *testing.T) {
	tx := MapPlaidTransaction(PlaidTransaction{
		ProviderItemID:        "pi_demo",
		ProviderTransactionID: "plaid_tx_1",
		ProviderAccountID:     "plaid_acc_1",
		Date:                  "2026-05-10",
		Amount:                12.34,
		Name:                  "Coffee",
		Currency:              "USD",
		Pending:               true,
	})

	if tx.AmountMinorUnits != -1234 {
		t.Fatalf("amount minor units = %d, want -1234", tx.AmountMinorUnits)
	}
	if !tx.Pending {
		t.Fatal("pending = false, want true")
	}
}

func TestMapPlaidCreditAccountNormalizesLiabilityBalanceToNegative(t *testing.T) {
	account := MapPlaidAccount(PlaidAccount{
		ProviderItemID:           "pi_demo",
		ProviderAccountID:        "plaid_credit_1",
		Name:                     "Credit Card",
		Type:                     "credit",
		Subtype:                  "credit card",
		CurrentBalance:           456.78,
		AvailableCredit:          floatPtr(1500),
		Currency:                 "USD",
	})

	if account.CurrentBalanceMinorUnits != -45678 {
		t.Fatalf("current balance minor units = %d, want -45678", account.CurrentBalanceMinorUnits)
	}
	if account.AvailableCreditMinorUnits == nil || *account.AvailableCreditMinorUnits != 150000 {
		t.Fatalf("available credit = %#v, want 150000", account.AvailableCreditMinorUnits)
	}
}

func TestMapBridgeTransactionKeepsPositiveCreditAndNegativeDebit(t *testing.T) {
	credit := MapBridgeTransaction(BridgeTransaction{
		ProviderItemID:        "pi_bridge",
		ProviderTransactionID: "bridge_tx_credit",
		ProviderAccountID:     "bridge_acc",
		Date:                  "2026-05-10",
		Amount:                25,
		Direction:             "credit",
		Description:           "Interest",
		Currency:              "USD",
	})
	debit := MapBridgeTransaction(BridgeTransaction{
		ProviderItemID:        "pi_bridge",
		ProviderTransactionID: "bridge_tx_debit",
		ProviderAccountID:     "bridge_acc",
		Date:                  "2026-05-10",
		Amount:                25,
		Direction:             "debit",
		Description:           "Payment",
		Currency:              "USD",
	})

	if credit.AmountMinorUnits != 2500 {
		t.Fatalf("credit amount = %d, want 2500", credit.AmountMinorUnits)
	}
	if debit.AmountMinorUnits != -2500 {
		t.Fatalf("debit amount = %d, want -2500", debit.AmountMinorUnits)
	}
}

func floatPtr(v float64) *float64 {
	return &v
}

