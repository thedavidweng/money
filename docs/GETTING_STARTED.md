# Getting Started

## Install

### macOS / Linux (Homebrew)

```bash
brew install thedavidweng/tap/money
```

### All platforms (Go)

Requires Go 1.25 or later:

```bash
go install github.com/thedavidweng/money/cmd/money@latest
```

Make sure `$GOPATH/bin` (usually `$HOME/go/bin`) is in your `PATH`.

### Pre-built binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/thedavidweng/money/releases), unzip the archive, and move the `money` binary to a directory in your `PATH`.

---

## First-time setup

Run the guided setup. It creates your local config directory, generates an encryption key, and initializes an encrypted SQLite database:

```bash
money setup
```

After setup, if no financial provider is configured yet, `money` walks you through an interactive wizard:

1. **Choose a provider** — select Plaid, Bridge, or skip for now.
2. **Configure credentials** — for Plaid, sign in to the Plaid Dashboard and fetch API keys automatically, or paste `client_id` and `secret` manually.
3. **Store credentials locally** — API keys are written to `.env`; config keeps explicit `env:` references.
4. **Repeat or finish** — configure additional providers, or skip and come back later with `money providers configure <provider>`.

Example interactive flow:

```text
$ money setup
Config:   /Users/you/.money/config.yaml
Secrets:  /Users/you/.money/.env
Database: /Users/you/.money/data/money.db
Encryption key: created
Database: ready

No providers are configured yet. To link financial institutions, you need at least one provider.

How do you want to configure Plaid?
> Sign in with Plaid Dashboard and fetch API keys automatically
  Paste client ID and secret manually
  Skip Plaid for now

plaid configured (2 credentials written).
  ✓ [Config] Config loaded from /Users/you/.money/config.yaml
  ✓ [Providers] plaid credentials present.
```

Plaid Dashboard login is best-effort because it follows Plaid CLI-compatible Dashboard behavior. Manual configuration remains the durable path:

```bash
money providers configure plaid --client-id <client-id> --secret <secret> --environment sandbox
```

### Try it without real credentials

Use demo mode to explore the CLI with bundled sample data:

```bash
money demo accounts list --json
money demo transactions list --json
money demo transactions search "coffee" --json
```

Demo mode is non-persistent and does not require provider credentials.

---

## Link an institution

Once a provider is configured, link your bank:

```bash
money link
```

This starts an institution-first flow: search for your bank, choose the provider that supports it, and authenticate.

Then sync data into your local encrypted database:

```bash
money sync
```

For local Plaid Sandbox testing without the browser Link flow, use:

```bash
money plaid sandbox link --products transactions --institution-id ins_56
```

After syncing, query your local data:

```bash
money accounts list --json
money transactions list --json
money transactions search "Costco" --json

# Investments and liabilities (synced via Plaid investments/liabilities products)
money investments holdings --json
money investments securities --json
money liabilities list --json

# Manage provider items
money items list
money items rename <id> "My Chase Savings"
```

---

## Pricing

`money` itself is free and open source under the MIT license. You bring your own provider credentials (BYOK). Provider costs depend on the provider you choose.

### Plaid

Plaid offers a free Trial plan for new teams. See [Plaid billing documentation](https://plaid.com/docs/account/billing/#trial-plans) for full details.

**Free Trial plans** are available to new Plaid teams (US/Canada only) created on or after April 15, 2026. You can create **10 connection Items** on a Trial plan:

- One bank login = **1 connection** (even if it contains multiple accounts).
- Same user at two different banks = **2 connections**.
- The 10 limit is on **connections**, not on API calls.
- You can make **unlimited API calls** against those up-to-10 Items.

After the Trial plan, you may upgrade to a paid Plaid plan if your usage exceeds the free tier.

### Bridge

Bridge pricing is separate and set by Bridge. Check [Bridge API pricing](https://dashboard.bridgeapi.io/) for current plans.

---

## Next steps

- Read [`docs/ARCHITECTURE.md`](ARCHITECTURE.md) to understand how data flows.
- Read [`docs/CONFIG.md`](CONFIG.md) for advanced configuration options.
- Run `money doctor` to check configuration health at any time.
