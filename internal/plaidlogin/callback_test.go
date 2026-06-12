package plaidlogin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCallbackHandlerAcceptsValidCodeOnce(t *testing.T) {
	server := NewCallbackServer("state-ok", time.Second)
	httpServer := httptest.NewServer(server.Handler())
	defer httpServer.Close()

	resp, err := http.Get(httpServer.URL + "/oauth/callback?code=auth-code&state=state-ok")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	result, err := server.Wait(context.Background())
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if result.Code != "auth-code" {
		t.Fatalf("Code = %q", result.Code)
	}

	resp, err = http.Get(httpServer.URL + "/oauth/callback?code=second&state=state-ok")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want %d", resp.StatusCode, http.StatusConflict)
	}
}

func TestCallbackHandlerRejectsOAuthErrorMissingCodeWrongStateAndWrongMethod(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		target      string
		status      int
		wantWaitErr bool // true if Wait should return immediately with an error
	}{
		{name: "oauth error", method: http.MethodGet, target: "/oauth/callback?error=access_denied&state=state-ok", status: http.StatusBadRequest, wantWaitErr: true},
		{name: "missing code", method: http.MethodGet, target: "/oauth/callback?state=state-ok", status: http.StatusBadRequest, wantWaitErr: true},
		{name: "wrong state", method: http.MethodGet, target: "/oauth/callback?code=auth-code&state=bad", status: http.StatusForbidden, wantWaitErr: true},
		{name: "wrong method", method: http.MethodPost, target: "/oauth/callback?code=auth-code&state=state-ok", status: http.StatusMethodNotAllowed, wantWaitErr: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := NewCallbackServer("state-ok", time.Second)
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(""))
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tc.status, rec.Body.String())
			}
			if tc.wantWaitErr {
				// Wait should return immediately with an error, not hang.
				waitCtx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				_, err := server.Wait(waitCtx)
				if err == nil {
					t.Fatal("Wait returned nil, want OAuth callback error")
				}
			}
		})
	}
}

func TestCallbackWaitTimesOut(t *testing.T) {
	server := NewCallbackServer("state-ok", 5*time.Millisecond)
	_, err := server.Wait(context.Background())
	if err == nil {
		t.Fatal("Wait succeeded, want timeout")
	}
}
