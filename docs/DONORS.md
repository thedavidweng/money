# Donor Notes

`money` is a clean rewrite. Donor repositories are used only for local inspection and product/architecture research.

Local donor clones may live under `donors/`. That directory is ignored by git to avoid vendoring unrelated histories, license-sensitive source, generated assets, or research notes into this project.

Do not commit donor source code. If implementation work is inspired by a donor, rewrite the implementation in this project and document the inspiration and license constraints here or in the README acknowledgements.

## Priority Rule

Ray Finance is the primary engineering donor for local schema, Provider sync, Plaid/Bridge behavior, imports, local annotations, and query fields. `monarchmoney-cli` is the primary CLI donor for command naming, JSON envelopes, stdout/stderr behavior, exit codes, safety gates, and agent ergonomics. If Ray Finance lacks a feature, do not create a first-version stable contract from Monarch alone unless `money` explicitly needs it.

## monarchmoney-cli

Use as the strongest donor for:

- Agent-friendly CLI shape.
- JSON envelopes.
- stdout/stderr separation.
- Predictable command names.
- Safety gates such as read-only, dry-run, and confirm.
- Documentation discipline for command changes.

Avoid copying:

- Monarch-only GraphQL assumptions into core.
- Remote Monarch as the canonical source of truth.
- Thin cache schema as the long-term local schema.

## ray-finance

Use as the strongest donor for:

- Plaid and Bridge sync lessons.
- Local configured-state checks for Providers: Ray treats Plaid as configured when local BYOK config has client ID plus secret, and Bridge as configured when local config has client ID plus client secret. This is local configuration state, not a bank/API-discovered property.
- Plaid CLI `20260507-4d1b0ca0` is discovery evidence for Plaid Dashboard OAuth bootstrap only: `plaid login` uses Dashboard OAuth, lists teams, and fetches API keys automatically. The Dashboard API endpoints are private Plaid CLI-compatible behavior, so `money` must fail clearly if the contract changes and keep manual `money providers configure plaid` available.
- Local database schema experiments.
- Apple Card import.
- Recategorization rules.
- Spending, cashflow, debt, recurring, and portfolio query ideas.

Avoid copying:

- Embedded AI advisor.
- Chat, prompts, model providers, AI memory, and AI audit log.
- Managed setup, billing, hosted proxy, and product-site assumptions.
- Runtime `ray` compatibility fallbacks.

## actual

Use as a donor for:

- Local-first budgeting model.
- Import/export ideas.
- Automation API and package separation.
- Budgeting terminology and account/category behavior.

Avoid copying:

- Required server assumptions for bank sync.
- App-first complexity before the CLI contracts are stable.
- Budget-product-specific workflows that do not serve a provider-neutral backend.

## maybe

Use as a donor for:

- Personal finance domain vocabulary.
- Account/transaction/net-worth product ideas.
- Money and currency modeling.
- Rules and transaction command-center concepts.

Avoid copying:

- Rails/PostgreSQL architecture.
- AGPL code unless the license decision explicitly allows it.
- Product scope that assumes a full web app first.
