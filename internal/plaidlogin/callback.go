package plaidlogin

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type CallbackResult struct {
	Code string
	Err  error
}

type CallbackServer struct {
	state   string
	timeout time.Duration
	results chan CallbackResult
	mu      sync.Mutex
	seen    bool
}

type LocalCallbackServer struct {
	Port   int
	server *http.Server
}

func NewCallbackServer(state string, timeout time.Duration) *CallbackServer {
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	return &CallbackServer{
		state:   state,
		timeout: timeout,
		results: make(chan CallbackResult, 1),
	}
}

func (s *CallbackServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", s.handleCallback)
	return mux
}

func (s *CallbackServer) StartLocal() (LocalCallbackServer, error) {
	listener, err := net.Listen("tcp", BindHost+":0")
	if err != nil {
		return LocalCallbackServer{}, err
	}
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return LocalCallbackServer{}, fmt.Errorf("unexpected callback listener address %s", listener.Addr())
	}
	server := &http.Server{Handler: s.Handler()}
	go func() {
		_ = server.Serve(listener)
	}()
	return LocalCallbackServer{Port: tcpAddr.Port, server: server}, nil
}

func (s LocalCallbackServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *CallbackServer) Wait(ctx context.Context) (CallbackResult, error) {
	waitCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	select {
	case result := <-s.results:
		if result.Err != nil {
			return result, result.Err
		}
		return result, nil
	case <-waitCtx.Done():
		return CallbackResult{}, waitCtx.Err()
	}
}

// sendOnce sends result to the channel if no result has been sent yet.
// Returns true if this call sent the result, false if a result was already sent.
func (s *CallbackServer) sendOnce(result CallbackResult) bool {
	s.mu.Lock()
	if s.seen {
		s.mu.Unlock()
		return false
	}
	s.seen = true
	s.mu.Unlock()
	s.results <- result
	return true
}

func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		s.sendOnce(CallbackResult{Err: Error{Code: ErrorPlaidDashboardLoginRejected, Message: "Plaid Dashboard callback received invalid HTTP method"}})
		return
	}
	if r.URL.Query().Get("state") != s.state {
		http.Error(w, "invalid state", http.StatusForbidden)
		s.sendOnce(CallbackResult{Err: Error{Code: ErrorPlaidDashboardLoginRejected, Message: "Plaid Dashboard callback has invalid state parameter"}})
		return
	}
	if oauthError := r.URL.Query().Get("error"); oauthError != "" {
		http.Error(w, oauthError, http.StatusBadRequest)
		s.sendOnce(CallbackResult{Err: Error{Code: ErrorPlaidDashboardLoginRejected, Message: "Plaid Dashboard login rejected: " + oauthError}})
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		s.sendOnce(CallbackResult{Err: Error{Code: ErrorPlaidDashboardLoginRejected, Message: "Plaid Dashboard callback missing authorization code"}})
		return
	}
	if !s.sendOnce(CallbackResult{Code: code}) {
		http.Error(w, "callback already received", http.StatusConflict)
		return
	}
	_, _ = fmt.Fprint(w, "ok")
}
