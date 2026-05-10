package contracts

import (
	"encoding/json"
	"testing"
)

func TestSuccessEnvelopeUsesStructuredDiagnosticsAndMetadata(t *testing.T) {
	env := NewSuccess("accounts.list", map[string]any{"accounts": []any{}})
	env.Meta.Demo = true
	env.Meta.Pagination = &Pagination{Limit: 25, Offset: 0, Total: ptr(0), HasMore: false}
	env.Warnings = append(env.Warnings, Warning{
		Code:     "NO_LINKED_PROVIDER_ITEMS",
		Message:  "No linked provider items found.",
		Category: CategoryConfig,
	})

	payload, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	var decoded struct {
		Warnings []Warning  `json:"warnings"`
		Errors   []APIError `json:"errors"`
		Meta     struct {
			Demo       bool        `json:"demo"`
			Pagination *Pagination `json:"pagination"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	if len(decoded.Warnings) != 1 {
		t.Fatalf("warnings length = %d, want 1", len(decoded.Warnings))
	}
	if decoded.Warnings[0].Code == "" || decoded.Warnings[0].Category == "" {
		t.Fatalf("warning is not structured: %#v", decoded.Warnings[0])
	}
	if len(decoded.Errors) != 0 {
		t.Fatalf("errors length = %d, want 0", len(decoded.Errors))
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

	if len(env.Errors) != 1 {
		t.Fatalf("errors length = %d, want 1", len(env.Errors))
	}
	err := env.Errors[0]
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
