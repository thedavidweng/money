package providers

import (
	"context"

	"github.com/thedavidweng/money/internal/config"
)

type Registry struct {
	providers map[string]Provider
}

func NewRegistry(cfg config.Config) Registry {
	return Registry{providers: map[string]Provider{
		"plaid":  newPlaidProvider(cfg.Providers["plaid"]),
		"bridge": newBridgeProvider(cfg.Providers["bridge"]),
	}}
}

func (r Registry) Get(name string) (Provider, bool) {
	provider, ok := r.providers[name]
	return provider, ok
}

type staticProvider struct {
	name           string
	cfg            config.ProviderConfig
	requiredFields []string
}

func newStaticProvider(name string, cfg config.ProviderConfig, requiredFields ...string) staticProvider {
	return staticProvider{name: name, cfg: cfg, requiredFields: requiredFields}
}

func (p staticProvider) Name() string {
	return p.name
}

func (p staticProvider) ValidateConfig(ctx context.Context) []ConfigDiagnostic {
	for _, field := range p.requiredFields {
		if p.cfg.Fields == nil || p.cfg.Fields[field] == "" {
			return []ConfigDiagnostic{{
				Code:     "PROVIDER_CREDENTIALS_MISSING",
				Message:  p.name + " credentials are missing.",
				Severity: "warn",
			}}
		}
	}
	return nil
}

func (p staticProvider) SearchInstitutions(ctx context.Context, query string) ([]Institution, error) {
	return nil, ErrProviderNotImplemented
}

func (p staticProvider) CreateLinkSession(ctx context.Context, request LinkRequest) (LinkSession, error) {
	return LinkSession{}, ErrProviderNotImplemented
}

func (p staticProvider) ExchangeLinkToken(ctx context.Context, session LinkSession, callback LinkCallback) (LinkedItem, error) {
	return LinkedItem{}, ErrProviderNotImplemented
}

func (p staticProvider) Sync(ctx context.Context, item ProviderItem, sink SyncSink) (SyncResult, error) {
	return SyncResult{}, ErrProviderNotImplemented
}
