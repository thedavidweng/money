# Use app-managed encrypted SQLite for the local store

Status: Accepted

`money` stores financial accounts, transactions, sync state, and provider credentials locally, so the SQLite database must be encrypted at rest from the first provider-backed version. The encryption behavior is app-managed local encryption inspired by Ray Finance's BYOK path, not a zero-knowledge master-password vault and not managed SaaS. The concrete Go implementation should use the best fit for `money`'s single-binary architecture and current security best practices rather than copying Ray's `libsql` details. Commands that need the store must fail explicitly when the configured local database key is missing or invalid; real financial data must not silently fall back to plaintext SQLite.

## Decision

The MVP store uses `github.com/ncruces/go-sqlite3` with its `database/sql` driver and `github.com/ncruces/go-sqlite3/vfs/adiantum` encrypted VFS.

Configuration supplies exactly 32 random bytes through `MONEY_DB_ENCRYPTION_KEY`, encoded as base64url without padding. Config loading decodes and validates the key before opening a real database. The store passes the key to SQLite immediately after opening the encrypted connection and never places it in logs, JSON output, or command-line arguments.

Demo mode uses in-memory SQLite without encryption, applies the same migrations, and seeds bundled fixtures. Demo never reads or writes real user financial data, so this is not a fallback for real stores.

## Rationale

- It keeps the distribution cgo-free and close to a single binary.
- It uses SQLite semantics directly, so migrations and query tests stay boring.
- It accepts app-managed key material without inventing a custom encryption layer.
- It supports the same in-memory SQLite path needed by demo and tests.
- It avoids shipping a native SQLCipher dependency for the first implementation.

## Consequences

- The first security claim is encryption at rest, not tamper-proof database authentication.
- Temporary storage must be configured carefully for real encrypted stores, including `PRAGMA temp_store = memory`.
- If platform testing shows this driver or VFS is not viable, the next candidate is a SQLCipher-backed driver with the native dependency documented before adoption. Do not downgrade to plaintext SQLite or a custom field-encryption-only design.
