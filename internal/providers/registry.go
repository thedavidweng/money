package providers

import (
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
