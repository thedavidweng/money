import type { APIRoute } from 'astro';

const siteUrl = 'https://thedavidweng.github.io/money';
const repo = 'https://github.com/thedavidweng/money';

export const prerender = true;

export const GET: APIRoute = async () => {
	const content = `# money

> A local-first, self-hostable personal finance backend for AI agents and power users.

## Overview

money pulls account and transaction data from user-configured financial providers (Plaid, Bridge, ...), stores it in a user-owned encrypted SQLite database, and exposes stable CLI + JSON contracts for external agents, scripts, and cron jobs.

It does not embed AI chat, model providers, hosted billing, telemetry, or a required long-running server. Your data stays local. Your agent owns the reasoning.

## Why money

Existing personal finance tools either lock data behind a paid SaaS, embed opinionated AI advisors, or assume a full web-app product shape. money is different:

- Local-first data ownership — your financial data lives in an encrypted file you control.
- Agent-friendly contracts — stable JSON envelopes that any AI agent, script, or automation can parse.
- Provider-neutral — Plaid, Bridge, MX, Finicity, CSV imports — providers are replaceable adapters.
- No server required — runs as a CLI on your laptop, in cron, or in CI.

## Key features

- Encrypted SQLite — financial data at rest in an encrypted local file you control. Not plaintext SQLite, not someone else's cloud.
- BYOK providers — Plaid, Bridge, and more as adapters. You bring credentials; no managed proxy or subscription.
- Stable JSON contracts — versioned envelopes, deterministic sorting and pagination — built for scripts, cron, and agents.
- CLI-first — human output by default; --json when you need parseable stdout. No web server required.
- Explicit sync boundary — read commands use local data only. Network I/O happens when you link or sync — not on every query.
- Demo mode — money demo ... runs against bundled non-persistent sample data — no credentials required.

## Quick start

### Step 1: Install

\`\`\`
brew install --cask thedavidweng/tap/money
\`\`\`

Or via Go:

\`\`\`
go install github.com/thedavidweng/money/cmd/money@latest
\`\`\`

### Step 2: Try without credentials

\`\`\`
money demo accounts list --json
money demo transactions search "coffee"
\`\`\`

### Step 3: Real sync (BYOK)

\`\`\`
money setup
money providers configure plaid --client-id ... --secret ... --environment sandbox
money link
money sync
money accounts list --json
money transactions search "Costco" --json
\`\`\`

## Architecture

Data flow: Provider -> Adapter -> Canonical records -> Encrypted SQLite -> Core query -> JSON contract -> Your agent

Read commands never call the network. Outbound provider traffic only happens when you explicitly link or sync.

Providers map into canonical records — accounts, transactions, categories, tags, recurring items — and land in encrypted SQLite. Every command returns a versioned JSON envelope with deterministic sorting and pagination.

## Available commands

- \`money setup\` — Initialize configuration and encrypted database
- \`money doctor\` — Check configuration and system health
- \`money accounts list\` — List financial accounts
- \`money accounts create-manual\` — Create a local manual account
- \`money transactions list\` — List transactions with filters
- \`money transactions search\` — Search transactions by text
- \`money categories list\` — List transaction categories
- \`money tags list\` — List transaction tags
- \`money recurring list\` — List recurring transactions
- \`money link\` — Link a financial institution
- \`money providers configure <provider>\` — Configure provider credentials
- \`money providers plaid link\` — Link a Plaid Provider Item
- \`money providers bridge link\` — Link a Bridge Provider Item
- \`money sync\` — Sync linked provider data
- \`money demo <command>\` — Run against non-persistent sample data

Read commands and sync support --json for machine-readable output. Manual write operations require --dry-run or --confirm.

## Design principles

1. CLI-first — human output by default, --json for automation.
2. Stable contracts — versioned JSON envelopes with deterministic sorting and pagination.
3. Explicit failures — no hidden fallbacks, no silent downgrades.
4. BYOK providers — bring your own credentials; no managed proxy or subscription.
5. Encrypted at rest — real financial data never touches plaintext SQLite.
6. Small deep modules — simple interfaces over pass-through wrappers.

## Documentation

- docs/PRD.md — Product requirements
- docs/ARCHITECTURE.md — Module boundaries and data flow
- docs/CONTRACTS.md — Current CLI JSON contracts
- docs/SCHEMA.md — Database schema contract
- docs/CONFIG.md — Configuration loading rules
- docs/ROADMAP.md — Development phases

## Contact

- GitHub Issues: ${repo}/issues
- Repository: ${repo}
- License: Apache 2.0

## Machine-Readable Endpoints

- Agent JSON: ${siteUrl}/agent
- LLMs.txt: ${siteUrl}/llms.txt
- Markdown homepage: ${siteUrl}/index.md
- Sitemap: ${siteUrl}/sitemap.xml
- robots.txt: ${siteUrl}/robots.txt
`;

	return new Response(content, {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
