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
	helper := NewPlaidLinkHelper(PlaidLinkHelperConfig{LinkToken: "link-token", State: "state", Timeout: time.Second})
	handler := helper.Handler()

	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/other", http.NoBody))
	if resp.Code != http.StatusNotFound {
		t.Fatalf("other path status = %d", resp.Code)
	}

	pageResp := httptest.NewRecorder()
	handler.ServeHTTP(pageResp, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	if pageResp.Code != http.StatusOK {
		t.Fatalf("page status = %d", pageResp.Code)
	}
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(pageResp.Body)
	body := buf.String()
	if !strings.Contains(body, "link-token") || !strings.Contains(body, "Plaid.create") {
		t.Fatalf("link page does not contain Plaid Link bootstrap: %s", body)
	}
}

func TestPlaidLinkHelperInvalidBodyPushesErrorCallback(t *testing.T) {
	helper := NewPlaidLinkHelper(PlaidLinkHelperConfig{LinkToken: "link-token", State: "state", Timeout: time.Second})
	handler := helper.Handler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(`{"invalid`))
	req.Host = "127.0.0.1"
	req.Header.Set("Origin", "http://127.0.0.1")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}

	callback, err := helper.Wait(context.Background())
	if err != nil {
		t.Fatalf("wait callback: %v", err)
	}
	if callback.Status != "error" || callback.Error.Code != "INVALID_CALLBACK_PAYLOAD" {
		t.Fatalf("expected body-decode error callback, got status=%q code=%q", callback.Status, callback.Error.Code)
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

	callback, err := helper.Wait(context.Background())
	if err != nil {
		t.Fatalf("wait callback: %v", err)
	}
	if callback.Status != "error" || callback.Error.Code != "INVALID_STATE" {
		t.Fatalf("expected state-mismatch error callback, got status=%q code=%q", callback.Status, callback.Error.Code)
	}
}

func TestPlaidLinkHelperAcceptsValidCallbackAfterStateMismatchIsConsumed(t *testing.T) {
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

	// Consume the error callback pushed by the state-mismatch path.
	_, _ = helper.Wait(context.Background())

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

func TestPlaidLinkHelperHandlesSuccessCancelAndLinkErrorPayloads(t *testing.T) {
	for name, payload := range map[string]string{
		"success": `{"status":"success","public_token":"public","state":"state","metadata":{"institution":{"institution_id":"ins_123","name":"Bank"}}}`,
		"cancel":  `{"status":"cancel","state":"state","metadata":{"link_session_id":"link-session"}}`,
		"error":   `{"status":"error","state":"state","error":{"error_type":"ITEM_ERROR","error_code":"INVALID_CREDENTIALS","error_message":"bad credentials","display_message":"try again"},"metadata":{"request_id":"req_123","link_session_id":"link-session"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			helper := NewPlaidLinkHelper(PlaidLinkHelperConfig{LinkToken: "link-token", State: "state", Timeout: time.Second})
			resp := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(payload))
			req.Header.Set("Origin", "http://127.0.0.1")
			req.Host = "127.0.0.1"
			helper.Handler().ServeHTTP(resp, req)
			if resp.Code != http.StatusOK {
				t.Fatalf("callback status = %d", resp.Code)
			}
			callback, err := helper.Wait(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if callback.Status != name {
				t.Fatalf("callback status = %q", callback.Status)
			}
			if name == "error" && (callback.Error.Code != "INVALID_CREDENTIALS" || callback.Metadata.RequestID != "req_123") {
				t.Fatalf("callback = %#v", callback)
			}
			if name == "cancel" && callback.Metadata.LinkSessionID != "link-session" {
				t.Fatalf("callback = %#v", callback)
			}
		})
	}
}

func TestPlaidLinkHelperRejectsWrongOriginAndDuplicateCallback(t *testing.T) {
	t.Run("bad origin pushes error callback", func(t *testing.T) {
		helper := NewPlaidLinkHelper(PlaidLinkHelperConfig{LinkToken: "link-token", State: "state", Timeout: time.Second})
		handler := helper.Handler()

		badOrigin := httptest.NewRecorder()
		badReq := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(`{"status":"cancel","state":"state"}`))
		badReq.Host = "127.0.0.1"
		badReq.Header.Set("Origin", "http://evil.test")
		handler.ServeHTTP(badOrigin, badReq)
		if badOrigin.Code != http.StatusForbidden {
			t.Fatalf("bad origin status = %d", badOrigin.Code)
		}

		callback, _ := helper.Wait(context.Background())
		if callback.Status != "error" || callback.Error.Code != "ORIGIN_VALIDATION_FAILED" {
			t.Fatalf("expected origin error callback, got status=%q code=%q", callback.Status, callback.Error.Code)
		}
	})

	t.Run("non-loopback origin pushes error callback", func(t *testing.T) {
		helper := NewPlaidLinkHelper(PlaidLinkHelperConfig{LinkToken: "link-token", State: "state", Timeout: time.Second})
		handler := helper.Handler()

		nonLoopback := httptest.NewRecorder()
		nonLoopbackReq := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(`{"status":"cancel","state":"state"}`))
		nonLoopbackReq.Host = "example.com"
		nonLoopbackReq.Header.Set("Origin", "http://example.com")
		handler.ServeHTTP(nonLoopback, nonLoopbackReq)
		if nonLoopback.Code != http.StatusForbidden {
			t.Fatalf("non-loopback origin status = %d", nonLoopback.Code)
		}

		callback, _ := helper.Wait(context.Background())
		if callback.Status != "error" || callback.Error.Code != "ORIGIN_VALIDATION_FAILED" {
			t.Fatalf("expected origin error callback, got status=%q code=%q", callback.Status, callback.Error.Code)
		}
	})

	t.Run("bare hostname matching Host header does not panic", func(t *testing.T) {
		helper := NewPlaidLinkHelper(PlaidLinkHelperConfig{LinkToken: "link-token", State: "state", Timeout: time.Second})
		handler := helper.Handler()

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(`{"status":"cancel","state":"state"}`))
		req.Host = "myhost"
		req.Header.Set("Origin", "http://myhost")
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("bare hostname status = %d", rec.Code)
		}

		callback, _ := helper.Wait(context.Background())
		if callback.Status != "error" || callback.Error.Code != "ORIGIN_VALIDATION_FAILED" {
			t.Fatalf("expected origin error callback, got status=%q code=%q", callback.Status, callback.Error.Code)
		}
	})

	t.Run("duplicate callback", func(t *testing.T) {
		helper := NewPlaidLinkHelper(PlaidLinkHelperConfig{LinkToken: "link-token", State: "state", Timeout: time.Second})
		handler := helper.Handler()

		first := httptest.NewRecorder()
		firstReq := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(`{"status":"cancel","state":"state"}`))
		firstReq.Host = "127.0.0.1"
		firstReq.Header.Set("Origin", "http://127.0.0.1")
		handler.ServeHTTP(first, firstReq)
		if first.Code != http.StatusOK {
			t.Fatalf("first status = %d", first.Code)
		}

		second := httptest.NewRecorder()
		secondReq := httptest.NewRequest(http.MethodPost, "/callback", strings.NewReader(`{"status":"cancel","state":"state"}`))
		secondReq.Host = "127.0.0.1"
		secondReq.Header.Set("Origin", "http://127.0.0.1")
		handler.ServeHTTP(second, secondReq)
		if second.Code != http.StatusConflict {
			t.Fatalf("second status = %d", second.Code)
		}
	})
}

func TestPlaidLinkHelperWaitTimesOut(t *testing.T) {
	helper := NewPlaidLinkHelper(PlaidLinkHelperConfig{LinkToken: "link-token", State: "state", Timeout: time.Millisecond})
	_, err := helper.Wait(context.Background())
	if err == nil {
		t.Fatal("expected timeout")
	}
}
