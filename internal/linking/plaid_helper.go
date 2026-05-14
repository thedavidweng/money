package linking

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/thedavidweng/money/internal/providers"
)

type PlaidLinkHelperConfig struct {
	LinkToken string
	State     string
	Timeout   time.Duration
}

type PlaidLinkHelper struct {
	linkToken string
	state     string
	timeout   time.Duration
	callback  chan providers.LinkCallback
}

func NewPlaidLinkHelper(cfg PlaidLinkHelperConfig) *PlaidLinkHelper {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	return &PlaidLinkHelper{
		linkToken: cfg.LinkToken,
		state:     cfg.State,
		timeout:   timeout,
		callback:  make(chan providers.LinkCallback, 1),
	}
}

func NewLinkState() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (h *PlaidLinkHelper) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleIndex)
	mux.HandleFunc("/callback", h.handleCallback)
	return mux
}

func (h *PlaidLinkHelper) StartTestServer() *httptest.Server {
	return httptest.NewServer(h.Handler())
}

type LocalPlaidLinkServer struct {
	URL    string
	server *http.Server
}

func (h *PlaidLinkHelper) StartLocalServer() (*LocalPlaidLinkServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	server := &http.Server{Handler: h.Handler()}
	go func() {
		_ = server.Serve(listener)
	}()
	return &LocalPlaidLinkServer{
		URL:    "http://" + listener.Addr().String(),
		server: server,
	}, nil
}

func (s *LocalPlaidLinkServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (h *PlaidLinkHelper) Wait(ctx context.Context) (providers.LinkCallback, error) {
	waitCtx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()
	select {
	case callback := <-h.callback:
		return callback, nil
	case <-waitCtx.Done():
		return providers.LinkCallback{}, waitCtx.Err()
	}
}

func (h *PlaidLinkHelper) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = plaidLinkPage.Execute(w, struct {
		LinkToken string
		State     string
	}{LinkToken: h.linkToken, State: h.state})
}

func (h *PlaidLinkHelper) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !validCallbackOrigin(r) {
		http.Error(w, "invalid origin", http.StatusForbidden)
		select {
		case h.callback <- providers.LinkCallback{
			Status: "error",
			Error:  providers.LinkError{Type: "ORIGIN_ERROR", Code: "ORIGIN_VALIDATION_FAILED", Message: "callback request failed origin validation"},
		}:
		default:
		}
		return
	}
	var payload struct {
		PublicToken string                 `json:"public_token"`
		State       string                 `json:"state"`
		Status      string                 `json:"status"`
		Metadata    providers.LinkMetadata `json:"metadata"`
		Error       providers.LinkError    `json:"error"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid callback", http.StatusBadRequest)
		return
	}
	if payload.State != h.state {
		http.Error(w, "invalid state", http.StatusForbidden)
		return
	}
	status := payload.Status
	if status == "" {
		status = "success"
	}
	callback := providers.LinkCallback{
		PublicToken: payload.PublicToken,
		State:       payload.State,
		Status:      status,
		Metadata:    payload.Metadata,
		Error:       payload.Error,
	}
	select {
	case h.callback <- callback:
	default:
		http.Error(w, "callback already received", http.StatusConflict)
		return
	}
	fmt.Fprint(w, "ok")
}

func validCallbackOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" {
		return false
	}
	if parsed.Host != r.Host {
		return false
	}
	host := parsed.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

var plaidLinkPage = template.Must(template.New("plaid-link").Parse(`<!doctype html>
<html>
<head><meta charset="utf-8"><title>money Plaid Link</title></head>
<body>
<script src="https://cdn.plaid.com/link/v2/stable/link-initialize.js"></script>
<script>
const handler = Plaid.create({
  token: {{printf "%q" .LinkToken}},
  onSuccess: function(public_token, metadata) {
    fetch("/callback", {
      method: "POST",
      headers: {"Content-Type": "application/json"},
      body: JSON.stringify({status: "success", public_token: public_token, state: {{printf "%q" .State}}, metadata: metadata})
    });
  },
  onExit: function(err, metadata) {
    fetch("/callback", {
      method: "POST",
      headers: {"Content-Type": "application/json"},
      body: JSON.stringify({status: err ? "error" : "cancel", state: {{printf "%q" .State}}, error: err, metadata: metadata})
    });
  }
});
handler.open();
</script>
</body>
</html>`))
