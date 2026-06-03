# money — Project Website

This is the [Astro](https://astro.build/)-based landing page for the `money` project. It is deployed to GitHub Pages at **https://thedavidweng.github.io/money/**.

## Development

```bash
cd website
bun install
bun run dev       # local preview at localhost:4321
bun run build     # output in website/dist/
bun run test      # build + asset verification
```

Requires [Bun](https://bun.sh/) 1.3+ and Node 20+.

## Deployment

The site is built and deployed automatically via GitHub Actions on push to `main` when files under `website/` change. Enable **GitHub Pages → GitHub Actions** in the repository settings if it is not live yet.
