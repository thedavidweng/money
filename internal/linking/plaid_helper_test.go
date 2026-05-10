package linking

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewLinkStateReturnsRandomBase64URLState(t *testing.T) {
	first, err := NewLinkState()
	if err != nil {
		t.Fatalf("new link state: %v", err)
	}
	second, err := NewLinkState()
	if err != nil {
		t.Fatalf("new link state: %v", err)
	}
	if first == second {
		t.Fatal("link state repeated")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(first)
	if err != nil {
		t.Fatalf("state is not raw base64url: %v", err)
	}
	if len(decoded) != 16 {
		t.Fatalf("decoded state length = %d, want 16", len(decoded))
	}
}

func TestPlaidLinkHelperServesOnlyLinkPageAndCallback(t *testing.T) {
	helper := NewPlaidLinkHelper(PlaidLinkHelperConfig{
		LinkToken: "link-token",
		State:     "state",
		Timeout:   time.Second,
	})

	handler := helper.Handler()
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("link page status = %d", resp.Code)
	}
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		t.Fatal(err)
	}
	body := buf.String()
	if !strings.Contains(body, "link-token") || !strings.Contains(body, "Plaid.create") {
		t.Fatalf("link page does not contain Plaid Link bootstrap: %s", body)
	}

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/accounts", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("unsupported path status = %d, want 404", missing.Code)
	}
}

func TestPlaidLinkHelperValidatesStateBeforeReturningCallback(t *testing.T) {
	helper := NewPlaidLinkHelper(PlaidLinkHelperConfig{
		LinkToken: "link-token",
		State:     "state",
		Timeout:   time.Second,
	})
	handler := helper.Handler()

	badPayload := strings.NewReader(`{"public_token":"public","state":"wrong"}`)
	badResp := httptest.NewRecorder()
	handler.ServeHTTP(badResp, httptest.NewRequest(http.MethodPost, "/callback", badPayload))
	if badResp.Code != http.StatusForbidden {
		t.Fatalf("bad callback status = %d, want 403", badResp.Code)
	}

	goodPayload := strings.NewReader(`{"public_token":"public","state":"state","metadata":{"institution":{"institution_id":"ins_123","name":"Bank"},"accounts":[{"id":"acc_1","name":"Checking","mask":"0000","type":"depository","subtype":"checking"}]}}`)
	goodResp := httptest.NewRecorder()
	handler.ServeHTTP(goodResp, httptest.NewRequest(http.MethodPost, "/callback", goodPayload))
	if goodResp.Code != http.StatusOK {
		t.Fatalf("good callback status = %d", goodResp.Code)
	}

	callback, err := helper.Wait(context.Background())
	if err != nil {
		t.Fatalf("wait callback: %v", err)
	}
	if callback.PublicToken != "public" || callback.State != "state" || callback.Metadata.Institution.ID != "ins_123" || len(callback.Metadata.Accounts) != 1 {
		encoded, _ := json.Marshal(callback)
		t.Fatalf("callback = %s", encoded)
	}
}
