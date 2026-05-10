package providers

import (
	"errors"
	"net"
)

var ErrProviderNotImplemented = errors.New("provider operation is not implemented")

type ProviderErrorKind string

const (
	ProviderErrorMissingCredentials ProviderErrorKind = "missing_credentials"
	ProviderErrorInvalidAuth        ProviderErrorKind = "invalid_authorization"
	ProviderErrorReconnectRequired  ProviderErrorKind = "reconnect_required"
	ProviderErrorRateLimit          ProviderErrorKind = "rate_limit"
	ProviderErrorNetwork            ProviderErrorKind = "network"
	ProviderErrorUnsupportedFeature ProviderErrorKind = "unsupported_feature"
	ProviderErrorAPI                ProviderErrorKind = "provider_api"
	ProviderErrorValidation         ProviderErrorKind = "validation"
	ProviderErrorInternal           ProviderErrorKind = "internal"
)

type ProviderAPIError struct {
	Provider   string
	StatusCode int
	Code       string
	Message    string
}

func (e ProviderAPIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

type ClassifiedProviderError struct {
	Provider  string
	Kind      ProviderErrorKind
	Code      string
	Message   string
	Retryable bool
}

func ClassifyProviderError(provider string, err error) ClassifiedProviderError {
	if err == nil {
		return ClassifiedProviderError{Provider: provider}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return ClassifiedProviderError{Provider: provider, Kind: ProviderErrorNetwork, Code: "NETWORK_ERROR", Message: err.Error(), Retryable: true}
	}

	var apiErr ProviderAPIError
	if errors.As(err, &apiErr) {
		return classifyAPIError(provider, apiErr)
	}

	return ClassifiedProviderError{Provider: provider, Kind: ProviderErrorInternal, Code: "PROVIDER_INTERNAL_ERROR", Message: err.Error(), Retryable: false}
}

func classifyAPIError(provider string, err ProviderAPIError) ClassifiedProviderError {
	if err.Provider != "" {
		provider = err.Provider
	}
	kind := ProviderErrorAPI
	retryable := false
	switch {
	case err.StatusCode == 401 || err.StatusCode == 403:
		kind = ProviderErrorInvalidAuth
	case err.StatusCode == 429:
		kind = ProviderErrorRateLimit
		retryable = true
	case err.StatusCode >= 500:
		kind = ProviderErrorAPI
		retryable = true
	case err.Code == "ITEM_LOGIN_REQUIRED" || err.Code == "USER_INPUT_REQUIRED":
		kind = ProviderErrorReconnectRequired
	case err.Code == "PRODUCT_NOT_SUPPORTED" || err.Code == "UNSUPPORTED_PRODUCT":
		kind = ProviderErrorUnsupportedFeature
	case err.StatusCode == 400 || err.StatusCode == 422:
		kind = ProviderErrorValidation
	}
	return ClassifiedProviderError{
		Provider:  provider,
		Kind:      kind,
		Code:      err.Code,
		Message:   err.Error(),
		Retryable: retryable,
	}
}
