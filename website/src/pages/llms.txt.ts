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

## Key features

- Encrypted SQLite — financial data at rest in an encrypted local file you control
- BYOK providers — bring your own Plaid / Bridge credentials; no managed proxy or subscription
- Stable JSON contracts — versioned envelopes, deterministic sorting and pagination
- CLI-first — human output by default; --json when you need parseable stdout
- Explicit sync boundary — read commands use local data only; network I/O only on link/sync
- Demo mode — try without credentials: money demo accounts list --json
- Apache 2.0 licensed, single Go binary, no telemetry

## Quick start

- Install: brew install --cask thedavidweng/tap/money
- Go install: go install github.com/thedavidweng/money/cmd/money@latest
- Demo: money demo accounts list --json
- Setup: money setup
- Sync: money link && money sync

## Contact

- GitHub Issues: ${repo}/issues
- Repository: ${repo}
- License: Apache 2.0

## Machine-Readable Endpoints

- Agent JSON: ${siteUrl}/agent
- Full content: ${siteUrl}/llms-full.txt
- Markdown homepage: ${siteUrl}/index.md
- Sitemap: ${siteUrl}/sitemap.xml
`;

	return new Response(content, {
		headers: { 'Content-Type': 'text/plain; charset=utf-8' },
	});
};
