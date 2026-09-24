# 0004: E2E-First Testing

Status: Accepted

Context: CLI tests asserted envelopes and tables via string matching. Store tests echoed CRUD lifecycles. E2E covered help text without artifact beyond demo JSON.

Decision: E2E is default with demo JSON, pagination, and filter artifacts. Isolated tests remain only for sync partial-failure, encrypted store, cursor dispatch, CSV and Monarch import edges, provider error classification, link origin and timeout guards, and migration schema with exact codes and DB artifacts. Coverage informational: patch 80 to 50, threshold 2 to 5.

Consequences: Deleted envelope, CRUD lifecycle, example, registry, link-request mirror, prompt fake, setup, rules, budgets, cli-shape, and E2E help tests. writeTestConfig and isPlaidLoginCode preserved as shared helpers.
