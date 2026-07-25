package contracts

import (
	"encoding/json"
	"testing"
)

func TestSuccessEnvelopeUsesStructuredDiagnosticsAndMetadata(t *testing.T) {
	env := NewSuccess("accounts.list", map[string]any{"accounts": []any{}})
	env.Meta.Demo = true
	env.Meta.Pagination = &Pagination{Limit: 25, Offset: 0, Total: ptr(0), HasMore: false}
	env.Meta.Warnings = append(env.Meta.Warnings, Warning{
		Code:     "NO_LINKED_PROVIDER_ITEMS",
		Message:  "No linked provider items found.",
		Category: CategoryConfig,
	})

	payload, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	var decoded struct {
		Error *APIError `json:"error"`
		Meta  struct {
			Demo       bool        `json:"demo"`
			Pagination *Pagination `json:"pagination"`
			Warnings   []Warning   `json:"warnings"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	if len(decoded.Meta.Warnings) != 1 {
		t.Fatalf("meta.warnings length = %d, want 1", len(decoded.Meta.Warnings))
	}
	if decoded.Meta.Warnings[0].Code == "" || decoded.Meta.Warnings[0].Category == "" {
		t.Fatalf("warning is not structured: %#v", decoded.Meta.Warnings[0])
	}
	if decoded.Error != nil {
		t.Fatalf("error = %#v, want nil on success", decoded.Error)
	}
	if !decoded.Meta.Demo {
		t.Fatal("meta.demo = false, want true")
	}
	if decoded.Meta.Pagination == nil {
		t.Fatal("meta.pagination is nil")
	}
}

func TestErrorEnvelopeIncludesCategoryAndRetryable(t *testing.T) {
	env := NewError("providers.plaid.link", "PROVIDER_AUTH_REQUIRED", "Plaid credentials are missing.", CategoryAuth, false)

	if env.Error == nil {
		t.Fatal("error is nil, want a single error object")
	}
	err := env.Error
	if err.Category != CategoryAuth {
		t.Fatalf("category = %q, want %q", err.Category, CategoryAuth)
	}
	if err.Retryable {
		t.Fatal("retryable = true, want false")
	}
}

func ptr[T any](v T) *T {
	return &v
}
