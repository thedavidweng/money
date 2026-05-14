package providers

import (
	"strings"
	"testing"

	plaid "github.com/plaid/plaid-go/v40/plaid"
)

func TestBuildPlaidLinkTokenCreateRequestUsesExplicitProductsAndCountries(t *testing.T) {
	request, err := BuildPlaidLinkTokenCreateRequest(PlaidLinkTokenRequestConfig{
		ClientName:    "money",
		Language:      "en",
		ClientUserID:  "local-user",
		Products:      []string{"transactions", "liabilities"},
		CountryCodes:  []string{"US", "CA"},
		RedirectURI:   "http://127.0.0.1:4000/callback",
		InstitutionID: "ins_123",
	})
	if err != nil {
		t.Fatalf("build request: %v", err)
	}

	if request.ClientName != "money" || request.Language != "en" {
		t.Fatalf("request identity = %q/%q", request.ClientName, request.Language)
	}
	if request.User == nil || request.User.ClientUserId != "local-user" {
		t.Fatalf("client user id = %#v", request.User)
	}
	if len(request.Products) != 2 || request.Products[0] != plaid.PRODUCTS_TRANSACTIONS || request.Products[1] != plaid.PRODUCTS_LIABILITIES {
		t.Fatalf("products = %#v", request.Products)
	}
	if len(request.CountryCodes) != 2 || request.CountryCodes[0] != plaid.COUNTRYCODE_US || request.CountryCodes[1] != plaid.COUNTRYCODE_CA {
		t.Fatalf("country codes = %#v", request.CountryCodes)
	}
	if request.RedirectUri == nil || *request.RedirectUri != "http://127.0.0.1:4000/callback" {
		t.Fatalf("redirect uri = %#v", request.RedirectUri)
	}
	if request.InstitutionId == nil || *request.InstitutionId != "ins_123" {
		t.Fatalf("institution id = %#v", request.InstitutionId)
	}
}

func TestBuildPlaidLinkTokenCreateRequestUsesConsentProductOptions(t *testing.T) {
	request, err := BuildPlaidLinkTokenCreateRequest(PlaidLinkTokenRequestConfig{
		ClientName:                  "money",
		Language:                    "en",
		ClientUserID:                "local-user",
		Products:                    []string{"transactions"},
		CountryCodes:                []string{"US"},
		AdditionalConsentedProducts: []string{"investments"},
		RequiredIfSupportedProducts: []string{"liabilities"},
		OptionalProducts:            []string{"auth"},
	})
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if len(request.AdditionalConsentedProducts) != 1 || request.AdditionalConsentedProducts[0] != plaid.PRODUCTS_INVESTMENTS {
		t.Fatalf("additional consented products = %#v", request.AdditionalConsentedProducts)
	}
	if len(request.RequiredIfSupportedProducts) != 1 || request.RequiredIfSupportedProducts[0] != plaid.PRODUCTS_LIABILITIES {
		t.Fatalf("required if supported products = %#v", request.RequiredIfSupportedProducts)
	}
	if len(request.OptionalProducts) != 1 || request.OptionalProducts[0] != plaid.PRODUCTS_AUTH {
		t.Fatalf("optional products = %#v", request.OptionalProducts)
	}
}

func TestBuildPlaidLinkTokenCreateRequestRejectsUnsupportedProduct(t *testing.T) {
	_, err := BuildPlaidLinkTokenCreateRequest(PlaidLinkTokenRequestConfig{
		ClientName:   "money",
		Language:     "en",
		ClientUserID: "local-user",
		Products:     []string{"balance"},
		CountryCodes: []string{"US"},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported Plaid Link product") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildPlaidLinkTokenCreateRequestRejectsUnsupportedConsentProduct(t *testing.T) {
	_, err := BuildPlaidLinkTokenCreateRequest(PlaidLinkTokenRequestConfig{
		ClientName:                  "money",
		Language:                    "en",
		ClientUserID:                "local-user",
		Products:                    []string{"transactions"},
		CountryCodes:                []string{"US"},
		AdditionalConsentedProducts: []string{"balance"},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported Plaid Link product") {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildPlaidLinkTokenCreateRequestRequiresExplicitCountries(t *testing.T) {
	_, err := BuildPlaidLinkTokenCreateRequest(PlaidLinkTokenRequestConfig{
		ClientName:   "money",
		Language:     "en",
		ClientUserID: "local-user",
		Products:     []string{"transactions"},
	})
	if err == nil || !strings.Contains(err.Error(), "country codes") {
		t.Fatalf("error = %v", err)
	}
}
