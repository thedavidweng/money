# Agent Instructions

不写兜底代码。检查过时/死代码。遵循奥卡姆剃刀。

## Agent skills

### Issue tracker

GitHub Issues.

### Triage labels

needs-triage, needs-info, ready-for-agent, ready-for-human, wontfix.

### Domain docs

`CONTEXT.md` + `docs/adr/` at repo root.

### Donor policy

See `.claude/skills/donor-policy.md` (load when touching `donors/`).

### Command contract

See `.claude/skills/command-contract.md` (load when changing CLI commands or JSON output).
