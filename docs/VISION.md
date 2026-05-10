# Vision

`money` is a local-first personal finance backend for external AI agents.

The immediate motivation is replacing a Monarch-dependent workflow without losing the ergonomics that made the Monarch CLI useful to agents. The long-term goal is not to build another finance chatbot. The product should be boring infrastructure: a user-owned store of financial data, a small set of deterministic finance primitives, and stable contracts that any agent can call.

## Product Identity

`money` is:

- A local financial data backend.
- A CLI-first automation surface.
- A provider-neutral sync system.
- A stable JSON contract layer for external agents.
- A self-hostable command surface for local automation.

`money` is not:

- An embedded AI advisor.
- A hosted subscription product.
- A budgeting app that requires a web server to be useful.
- A Plaid-only client.
- A Monarch clone.

## Core Bet

The agent workflow should depend on stable finance primitives, not on a specific data source.

Today the data source may be Monarch. Tomorrow it may be Plaid, MX, Finicity, a CSV importer, or a local manual account. The agent should still ask for accounts, transactions, budgets, rules, recurring bills, net worth, and cashflow using the same command semantics.
