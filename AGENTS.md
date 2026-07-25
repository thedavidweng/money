# AGENTS.md

Instructions for every coding agent working in this repo. This is the only agent-instruction file — never create CLAUDE.md, .cursorrules, .windsurfrules, .clinerules, GEMINI.md, or any other per-tool variant (CI fails the build if one appears).

## Code principles

1. Self-explanatory code, no comments. Write a comment only for a constraint the code cannot express (a protocol quirk, a required ordering, a spec reference). Decision rationale goes in `docs/adr/`, conventions go here — never in code. CI enforces a comment budget of 5% of non-test lines.
2. Fewest lines that stay clear. No fallback code, no speculative features, no dead code, no abstraction with a single caller. Delete before you add.
3. Code and docs move together: a change to commands, flags, output, or behavior updates the cobra help, COMMANDS.md, and JSON_SCHEMA.md in the same change.

## Verification

`mise run check` (fmt + build + test + lint + conventions) must pass before every push. CI runs the same gates.

## Decisions

Architectural decisions live in `docs/adr/NNNN-slug.md` (Status / Context / Decision / Consequences). A new dependency, a new abstraction, or a change to the public JSON/exit-code contract requires an ADR in the same change.

## Domain

`CONTEXT.md` at the repo root is the domain glossary.

## Issues

GitHub Issues via `gh`. Labels: needs-triage, needs-info, ready-for-agent, ready-for-human, wontfix.

## Donor policy

Donor repositories under `donors/` are local reference material only. See `docs/donor-policy.md` for the license-compatibility rules; load it when touching `donors/`.
