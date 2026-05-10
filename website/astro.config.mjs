// @ts-check
import { defineConfig } from 'astro/config';

// GitHub Project Pages: https://thedavidweng.github.io/money/
export default defineConfig({
  site: 'https://thedavidweng.github.io',
  base: '/money/',
  output: 'static',
  devToolbar: {
    enabled: false,
  },
});
