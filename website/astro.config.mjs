// @ts-check
import { defineConfig } from 'astro/config';

import sitemap from '@astrojs/sitemap';

// GitHub Project Pages: https://thedavidweng.github.io/money/
export default defineConfig({
  site: 'https://thedavidweng.github.io',
  base: '/money/',
  publicDir: '../public',
  output: 'static',

  devToolbar: {
    enabled: false,
  },

  integrations: [sitemap()],
});