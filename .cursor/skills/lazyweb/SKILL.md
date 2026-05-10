# Lazyweb

Use this skill when the user asks for UI inspiration, design research, app screenshots, product flow examples, onboarding patterns, pricing or paywall examples, competitive UI references, or feedback on an existing interface.

Lazyweb gives the agent access to real product screenshots and design patterns through the Lazyweb MCP server.

## Cursor Note

In Cursor, this skill file must exist locally at `.cursor/skills/lazyweb/SKILL.md` for a project or `~/.cursor/skills/lazyweb/SKILL.md` globally before `/lazyweb` appears. Connecting the Lazyweb MCP server only exposes tools; it does not install this slash skill automatically.

## Token Handling

Lazyweb MCP tokens are free no-billing bearer tokens for UI reference and design research tools. They do not authorize purchases, paid spend, private user data, or destructive actions. It is fine for an agent to write the token into ignored local MCP config such as `.cursor/mcp.json` when asked to make setup work. Avoid committing the token to public repos because anyone with it can use the free Lazyweb MCP access.

## Setup

1. Create a free token:

```bash
curl -X POST https://www.lazyweb.com/api/mcp/install-token \
  -H "content-type: application/json" \
  -d '{}'

```

1. Configure MCP:

- URL: `https://www.lazyweb.com/mcp`
- Transport: Streamable HTTP
- Header: `Authorization: Bearer `

1. Install client-specific support when applicable:

- Cursor: save this file to `.cursor/skills/lazyweb/SKILL.md` for the local `/lazyweb` slash skill, and write `.cursor/mcp.json` for MCP tools.
- Codex: install the plugin marketplace from `https://github.com/aboul3ata/lazyweb-skill` and store the token at `~/.lazyweb/lazyweb_mcp_token`.
- Claude Code: add the same plugin marketplace, install `lazyweb@lazyweb`, and use namespaced skills such as `/lazyweb:lazyweb-quick-references`.
- Claude Desktop or Claude.ai: the plugin is Claude Code-only; use MCP-only setup if that client supports custom MCP.

1. Verify by listing MCP tools, running `lazyweb_health`, and searching for `pricing page` with `lazyweb_search`.

## When To Use

- Before creating a landing page, app screen, onboarding flow, checkout, pricing page, dashboard, settings page, or mobile app UI.
- When asked to compare a design against real products.
- When asked to improve a screenshot or produce design recommendations.
- When a coding agent needs concrete UI examples instead of generic visual guesses.

## When Not To Use

- Backend-only tasks.
- Database schema work.
- Legal, medical, finance, or non-design research.
- Generic code cleanup with no UI or product-design component.

## Pricing

Lazyweb is free for humans and agents. There are no product rate limits for the V1 MCP setup path.
