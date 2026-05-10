# Agent Instructions

遵循奥卡姆剃刀和第一性原理，不写兜底代码，检查代码是否有过时/死代码。

This project is a local-first finance backend for external AI agents. Keep the core boring and deterministic.

## Project Rules

- Do not embed AI chat, model providers, conversation memory, hosted billing, telemetry, or required web-server behavior in the core.
- Keep provider-specific behavior behind provider adapters.
- Keep JSON contracts stable and tested before expanding command scope.
- Do not silently fall back to old config paths, old provider names, or donor project behavior.
- Do not commit `donors/`; it is local reference material only.
- Prefer small deep modules with simple interfaces over many pass-through wrappers.

## Donor Policy

Use donor repositories for reference:

- `donors/monarchmoney-cli`: CLI contract and safety design.
- `donors/ray-finance`: provider sync and local data lessons.
- `donors/actual`: local-first budgeting and automation lessons.
- `donors/maybe`: finance product/domain modeling.

Copying code from donors requires checking license compatibility first. Maybe is AGPL, so do not copy code from it unless the project intentionally adopts a compatible license.

## Command Contract Discipline

When changing CLI commands, flags, JSON output structure, or command behavior, update:

- Command help.
- `docs/PRD.md` if product requirements change.
- `docs/ARCHITECTURE.md` if module boundaries change.
- Contract tests once the command is stable.
