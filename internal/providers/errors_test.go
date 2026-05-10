package providers

import (
	"errors"
	"net"
	"testing"
)

func TestClassifyProviderErrorMapsNetworkErrors(t *testing.T) {
	classified := ClassifyProviderError("plaid", &net.DNSError{Err: "lookup failed"})

	if classified.Kind != ProviderErrorNetwork || !classified.Retryable {
		t.Fatalf("classified = %#v", classified)
	}
}

func TestClassifyProviderErrorMapsReconnectRequired(t *testing.T) {
	classified := ClassifyProviderError("plaid", ProviderAPIError{
		Provider: "plaid",
		Code:     "ITEM_LOGIN_REQUIRED",
		Message:  "Item requires relink.",
	})

	if classified.Kind != ProviderErrorReconnectRequired || classified.Retryable {
		t.Fatalf("classified = %#v", classified)
	}
}

func TestClassifyProviderErrorMapsRateLimit(t *testing.T) {
	classified := ClassifyProviderError("bridge", ProviderAPIError{
		Provider:   "bridge",
		StatusCode: 429,
		Code:       "rate_limit",
		Message:    "Rate limited.",
	})

	if classified.Kind != ProviderErrorRateLimit || !classified.Retryable {
		t.Fatalf("classified = %#v", classified)
	}
}

func TestClassifyProviderErrorKeepsUnknownInternal(t *testing.T) {
	classified := ClassifyProviderError("bridge", errors.New("unexpected"))

	if classified.Kind != ProviderErrorInternal || classified.Provider != "bridge" {
		t.Fatalf("classified = %#v", classified)
	}
}

