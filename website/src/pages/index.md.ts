import type { APIRoute } from 'astro';

const siteUrl = 'https://thedavidweng.github.io/money';
const repo = 'https://github.com/thedavidweng/money';

export const prerender = true;

export const GET: APIRoute = async () => {
	const content = `# money

A local-first, self-hostable personal finance backend for AI agents and power users.

[CLI] · [Local-first] · [For AI agents] · [v0.x pre-release]

## Local-first finance data your agent can rely on.

Pull accounts and transactions from providers you configure, store them in encrypted SQLite, and automate with stable CLI + JSON contracts. No embedded AI, no hosted ledger, no long-running server required.

- ✓ MIT licensed
- ✓ Single Go binary
- ✓ No telemetry

## Quick start

\`\`\`
brew install thedavidweng/tap/money

# Try with sample data (no credentials needed)
money demo accounts list --json
money demo transactions search "coffee"

# Then set up for real
money setup
money link
money sync
\`\`\`

## Features

- **Encrypted SQLite**: Financial data at rest in an encrypted local file you control.
- **BYOK providers**: Plaid, Bridge, and more as adapters. You bring credentials.
- **Stable JSON contracts**: Versioned envelopes, deterministic sorting and pagination.
- **CLI-first**: Human output by default; \`--json\` when you need parseable stdout.
- **Explicit sync boundary**: Read commands use local data only.
- **Demo mode**: \`money demo …\` runs against bundled sample data.

## Documentation

- [Product Requirements](${repo}/blob/main/docs/PRD.md)
- [Architecture](${repo}/blob/main/docs/ARCHITECTURE.md)
- [Database Schema](${repo}/blob/main/docs/SCHEMA.md)
- [Configuration](${repo}/blob/main/docs/CONFIG.md)
- [Roadmap](${repo}/blob/main/docs/ROADMAP.md)
- [Contributing](${repo}/blob/main/CONTRIBUTING.md)

## Contact

- GitHub: ${repo}
- Issues: ${repo}/issues
`;

	return new Response(content, {
		headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
	});
};
