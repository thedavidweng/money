---
name: command-contract
description: Checklist for CLI contract changes — run when modifying commands, flags, or JSON output
---

# Command Contract Discipline

When changing CLI commands, flags, JSON output structure, or command behavior, verify these are updated:

1. **Command help** — the cobra command's Short/Long/Example text
2. **`docs/PRD.md`** — if product requirements changed
3. **`docs/ARCHITECTURE.md`** — if module boundaries changed
4. **Contract tests** — once the command is stable

This is a checklist, not a blocker. If the change is internal-only (refactor without behavior change), skip.
