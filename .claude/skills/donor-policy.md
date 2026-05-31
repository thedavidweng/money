---
name: donor-policy
description: Rules for working with donor/reference repositories in donors/
---

# Donor Policy

Use donor repositories for reference:

- `donors/monarchmoney-cli`: CLI contract and safety design.
- `donors/ray-finance`: provider sync and local data lessons.
- `donors/actual`: local-first budgeting and automation lessons.
- `donors/maybe`: finance product/domain modeling.

## Rules

- Copying code from donors requires checking license compatibility first.
- **Maybe is AGPL** — do not copy code from it unless the project intentionally adopts a compatible license.
- Do not commit `donors/`; it is local reference material only.
- When using donor code as reference, adapt the pattern to this project's types and interfaces rather than copying verbatim.
