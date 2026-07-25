package providers

import (
	"fmt"
	"strings"

	plaid "github.com/plaid/plaid-go/v40/plaid"
)

type PlaidLinkTokenRequestConfig struct {
	ClientName                  string
	Language                    string
	ClientUserID                string
	Products                    []string
	CountryCodes                []string
	RedirectURI                 string
	InstitutionID               string
	AdditionalConsentedProducts []string
	RequiredIfSupportedProducts []string
	OptionalProducts            []string
}

func BuildPlaidLinkTokenCreateRequest(cfg *PlaidLinkTokenRequestConfig) (plaid.LinkTokenCreateRequest, error) {
	if cfg.ClientName == "" {
		return plaid.LinkTokenCreateRequest{}, fmt.Errorf("plaid client name is required")
	}
	if cfg.Language == "" {
		return plaid.LinkTokenCreateRequest{}, fmt.Errorf("plaid Link language is required")
	}
	if cfg.ClientUserID == "" {
		return plaid.LinkTokenCreateRequest{}, fmt.Errorf("plaid client user ID is required")
	}
	if len(cfg.Products) == 0 {
		return plaid.LinkTokenCreateRequest{}, fmt.Errorf("plaid Link products are required")
	}
	if len(cfg.CountryCodes) == 0 {
		return plaid.LinkTokenCreateRequest{}, fmt.Errorf("plaid Link country codes are required")
	}

	countries, err := plaidCountryCodes(cfg.CountryCodes)
	if err != nil {
		return plaid.LinkTokenCreateRequest{}, err
	}
	products, err := plaidLinkProducts(cfg.Products)
	if err != nil {
		return plaid.LinkTokenCreateRequest{}, err
	}
	additionalConsentedProducts, err := plaidLinkProducts(cfg.AdditionalConsentedProducts)
	if err != nil {
		return plaid.LinkTokenCreateRequest{}, err
	}
	requiredIfSupportedProducts, err := plaidLinkProducts(cfg.RequiredIfSupportedProducts)
	if err != nil {
		return plaid.LinkTokenCreateRequest{}, err
	}
	optionalProducts, err := plaidLinkProducts(cfg.OptionalProducts)
	if err != nil {
		return plaid.LinkTokenCreateRequest{}, err
	}

	request := plaid.NewLinkTokenCreateRequest(cfg.ClientName, cfg.Language, countries)
	request.User = plaid.NewLinkTokenCreateRequestUser(cfg.ClientUserID)
	request.SetProducts(products)
	if len(additionalConsentedProducts) > 0 {
		request.SetAdditionalConsentedProducts(additionalConsentedProducts)
	}
	if len(requiredIfSupportedProducts) > 0 {
		request.SetRequiredIfSupportedProducts(requiredIfSupportedProducts)
	}
	if len(optionalProducts) > 0 {
		request.SetOptionalProducts(optionalProducts)
	}
	if cfg.RedirectURI != "" {
		request.SetRedirectUri(cfg.RedirectURI)
	}
	if cfg.InstitutionID != "" {
		request.SetInstitutionId(cfg.InstitutionID)
	}
	return *request, nil
}

func plaidLinkProducts(values []string) ([]plaid.Products, error) {
	products := make([]plaid.Products, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		product := plaid.Products(strings.TrimSpace(value))
		switch product {
		case plaid.PRODUCTS_AUTH, plaid.PRODUCTS_TRANSACTIONS, plaid.PRODUCTS_INVESTMENTS, plaid.PRODUCTS_LIABILITIES:
			products = append(products, product)
		default:
			return nil, fmt.Errorf("unsupported Plaid Link product %q", value)
		}
	}
	return products, nil
}

func plaidCountryCodes(values []string) ([]plaid.CountryCode, error) {
	countries := make([]plaid.CountryCode, 0, len(values))
	for _, value := range values {
		country := plaid.CountryCode(strings.ToUpper(strings.TrimSpace(value)))
		if !country.IsValid() {
			return nil, fmt.Errorf("unsupported Plaid country code %q", value)
		}
		countries = append(countries, country)
	}
	return countries, nil
}
