import assert from 'node:assert/strict';
import { existsSync, readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';

const projectRoot = fileURLToPath(new URL('..', import.meta.url));
const distRoot = new URL('../dist/', import.meta.url);
const websitePublicDir = new URL('../public/', import.meta.url);

function assertBuiltAsset(pathname) {
	const relativePath = pathname.replace('/money/', '');
	assert.ok(
		existsSync(new URL(relativePath, distRoot)),
		`${pathname} is referenced by built HTML but missing from dist/`,
	);
}

test('built pages only reference public assets that exist in dist', () => {
	const pages = ['index.html', '404.html'];
	for (const page of pages) {
		const html = readFileSync(new URL(`../dist/${page}`, import.meta.url), 'utf8');
		for (const match of html.matchAll(/(?:href|src)="(\/money\/[^"#?]+)"/g)) {
			assertBuiltAsset(match[1]);
		}
	}
});

test('website uses the root public directory as its single media source', () => {
	const requiredAssets = ['favicon.png', 'Golden-Toad-logo.webp'];
	for (const asset of requiredAssets) {
		assert.ok(existsSync(join(projectRoot, '..', 'public', asset)), `public/${asset} must exist`);
		assert.ok(existsSync(new URL(asset, distRoot)), `dist/${asset} must be copied from public/`);
	}

	if (existsSync(websitePublicDir)) {
		const duplicateMedia = readdirSync(websitePublicDir).filter((name) => /\.(avif|ico|png|svg|webp)$/i.test(name));
		assert.deepEqual(duplicateMedia, [], 'website/public must not keep duplicate media assets');
	}
});
