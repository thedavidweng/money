import type { APIRoute } from 'astro';

const siteUrl = 'https://thedavidweng.github.io/money';
const repo = 'https://github.com/thedavidweng/money';

export const prerender = true;

export const GET: APIRoute = async () => {
	const agent = {
		identity: {
			name: 'money',
			description: 'A local-first, self-hostable personal finance backend for AI agents and power users.',
			url: siteUrl,
			repository: repo,
			license: 'Apache-2.0',
			version: '0.x (pre-release)',
		},
		services: [
			{
				name: 'CLI tool',
				description: 'Command-line interface for personal finance data management',
				commands: [
					'accounts list',
					'accounts create-manual',
					'transactions list',
					'transactions search',
					'categories list',
					'tags list',
					'recurring list',
					'link',
					'sync',
					'setup',
					'doctor',
					'demo',
				],
				interface: 'CLI + JSON contracts',
				outputFormat: 'Versioned JSON envelope with meta, data, errors',
			},
		],
		content: {
			overview: 'money pulls account and transaction data from user-configured financial providers, stores it in a user-owned encrypted SQLite database, and exposes stable CLI + JSON contracts for external agents, scripts, and cron jobs.',
			features: [
				'Encrypted SQLite — data at rest in an encrypted local file',
				'BYOK providers — Plaid, Bridge, and more as replaceable adapters',
				'Stable JSON contracts — versioned envelopes, deterministic ordering',
				'CLI-first — human output by default, --json for automation',
				'Explicit sync boundary — read uses local data only',
				'Demo mode — try without credentials against sample data',
				'Apache 2.0 licensed, single Go binary, no telemetry',
			],
			documentation: [
				{ name: 'PRD', url: `${repo}/blob/main/docs/PRD.md` },
				{ name: 'Architecture', url: `${repo}/blob/main/docs/ARCHITECTURE.md` },
				{ name: 'Contracts', url: `${repo}/blob/main/docs/CONTRACTS.md` },
				{ name: 'Schema', url: `${repo}/blob/main/docs/SCHEMA.md` },
				{ name: 'Config', url: `${repo}/blob/main/docs/CONFIG.md` },
				{ name: 'Roadmap', url: `${repo}/blob/main/docs/ROADMAP.md` },
			],
		},
		contact: {
			issues: `${repo}/issues`,
			repository: repo,
		},
		meta: {
			schema: '1.0.0',
			generated: new Date().toISOString(),
			deploy: 'GitHub Pages (GitHub Actions)',
			framework: 'Astro 6',
		},
	};

	return new Response(JSON.stringify(agent, null, 2), {
		headers: { 'Content-Type': 'application/json; charset=utf-8' },
	});
};
