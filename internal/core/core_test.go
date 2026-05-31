package core

import (
	"testing"
)

func TestFormatMinorUnits(t *testing.T) {
	tests := []struct {
		minor    int64
		currency string
		want     string
	}{
		{0, "USD", "0.00"},
		{100, "USD", "1.00"},
		{-100, "USD", "-1.00"},
		{12345, "USD", "123.45"},
		{-12345, "USD", "-123.45"},
		{1, "USD", "0.01"},
		{-1, "USD", "-0.01"},
		{99999, "USD", "999.99"},
	}
	for _, tt := range tests {
		got := FormatMinorUnits(tt.minor, tt.currency)
		if got != tt.want {
			t.Errorf("FormatMinorUnits(%d, %q) = %q, want %q", tt.minor, tt.currency, got, tt.want)
		}
	}
}

func TestParseUnsignedDecimalMinorUnits(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"0", 0, false},
		{"1.00", 100, false},
		{"1", 100, false},
		{"123.45", 12345, false},
		{"0.01", 1, false},
		{"1,000.50", 100050, false},
		{"0.1", 10, false},
		{"10.", 1000, false},
		{"", 0, true},
		{"-1.00", 0, true},
		{"+1.00", 0, true},
		{"abc", 0, true},
		{"1.234", 0, true},
		{"1.2a", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseUnsignedDecimalMinorUnits(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseUnsignedDecimalMinorUnits(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseUnsignedDecimalMinorUnits(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestSignedManualBalance(t *testing.T) {
	tests := []struct {
		accountType string
		minor       int64
		wantSign    int64
		wantDir     string
		wantErr     bool
	}{
		{"depository", 1000, 1000, "increases", false},
		{"investment", 1000, 1000, "increases", false},
		{"property", 1000, 1000, "increases", false},
		{"vehicle", 1000, 1000, "increases", false},
		{"other_asset", 1000, 1000, "increases", false},
		{"credit", 1000, -1000, "decreases", false},
		{"loan", 1000, -1000, "decreases", false},
		{"other_liability", 1000, -1000, "decreases", false},
		{"unknown", 1000, 0, "", true},
	}
	for _, tt := range tests {
		gotSign, gotDir, err := SignedManualBalance(tt.accountType, tt.minor)
		if (err != nil) != tt.wantErr {
			t.Errorf("SignedManualBalance(%q, %d) error = %v, wantErr %v", tt.accountType, tt.minor, err, tt.wantErr)
			continue
		}
		if gotSign != tt.wantSign || gotDir != tt.wantDir {
			t.Errorf("SignedManualBalance(%q, %d) = (%d, %q), want (%d, %q)", tt.accountType, tt.minor, gotSign, gotDir, tt.wantSign, tt.wantDir)
		}
	}
}

func TestNewLocalID(t *testing.T) {
	id1, err := NewLocalID("tx_")
	if err != nil {
		t.Fatalf("NewLocalID: %v", err)
	}
	if len(id1) < 5 {
		t.Errorf("NewLocalID too short: %q", id1)
	}
	if id1[:3] != "tx_" {
		t.Errorf("NewLocalID prefix = %q, want tx_", id1[:3])
	}
	id2, err := NewLocalID("tx_")
	if err != nil {
		t.Fatalf("NewLocalID: %v", err)
	}
	if id1 == id2 {
		t.Errorf("NewLocalID produced duplicates: %q", id1)
	}
}

func TestOnlyDigits(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"123", true},
		{"0", true},
		{"", true},
		{"12a3", false},
		{"12.3", false},
		{"-1", false},
	}
	for _, tt := range tests {
		got := onlyDigits(tt.input)
		if got != tt.want {
			t.Errorf("onlyDigits(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestMinorUnitsToDecimal(t *testing.T) {
	tests := []struct {
		minor int64
		want  string
	}{
		{0, "0.00"},
		{100, "1.00"},
		{-100, "-1.00"},
		{12345, "123.45"},
		{1, "0.01"},
		{-1, "-0.01"},
	}
	for _, tt := range tests {
		got := FormatMinorUnits(tt.minor, "USD")
		if got != tt.want {
			t.Errorf("FormatMinorUnits(%d) = %q, want %q", tt.minor, got, tt.want)
		}
	}
}
