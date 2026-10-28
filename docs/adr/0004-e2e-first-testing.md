# 0004: E2E-First Testing

Status: Accepted

Context: CLI tests asserted envelopes and tables via string matching. Store tests echoed CRUD lifecycles. E2E covered help text without artifact beyond demo JSON.

Decision: E2E is default with demo JSON, pagination, and filter artifacts. Isolated tests remain only for sync partial-failure, encrypted store, cursor dispatch, CSV and Monarch import edges, provider error classification, link origin and timeout guards, and migration schema with exact codes and DB artifacts. Coverage informational: patch 80 to 50, threshold 2 to 5.

Consequences: Deleted envelope, CRUD lifecycle, example, registry, link-request mirror, prompt fake, setup, rules, budgets, cli-shape, and E2E help tests. writeTestConfig and isPlaidLoginCode preserved as shared helpers.

2026-10-27: Removed the schema-document text comparison, the command-help allowlist, the CSV and Monarch `Name()` echoes, the default-registry membership list, the `ImportError` format echo, and the loopback-host constant echo. Store tests still exercise migrations through SQLite. `TestRegistry` still checks register/get. `TestBuildAuthURLUsesPlaidCLICompatiblePKCE` still checks the redirect URI host. CLI behavior tests cover commands without a second command inventory.
