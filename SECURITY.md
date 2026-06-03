# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in `money`, please report it responsibly:

1. **Do not open a public GitHub issue.**
2. Email the maintainer at the address listed in the repository profile, or use [GitHub's private vulnerability reporting](https://github.com/thedavidweng/money/security/advisories/new).
3. Include a description of the vulnerability, steps to reproduce, and the potential impact.
4. You should receive an acknowledgement within 7 days.

## Scope

`money` handles sensitive data including financial credentials and an encrypted local database. The following are in scope:

- Credential leakage (API keys, access tokens, encryption keys)
- Bypass of the encrypted SQLite store (Adiantum VFS)
- Unauthorized data access through CLI commands
- Injection or data corruption via provider adapters or import sources
- Secrets written to plaintext (config files, logs, stderr, JSON output)

## Design Decisions

- **Encrypted at rest.** The SQLite database uses [Adiantum](https://github.com/AcademySoftwareFoundation/adiantum) VFS encryption. The encryption key is a 32-byte random value stored in the user's `.env` file with `0600` permissions.
- **No telemetry.** `money` does not phone home, embed analytics, or send data to any server other than the user-configured financial providers.
- **No server.** There is no long-running process, no listening ports, and no remote attack surface from the application itself.
- **BYOK credentials.** Provider API keys are the user's responsibility. `money` stores them locally and never transmits them to any party other than the configured provider's API.

## Known Limitations

- The Adiantum VFS provides encryption at rest (confidentiality), not database-level tamper detection (integrity/authentication). See [ADR-0001](docs/adr/0001-encrypted-sqlite-store.md) for the full rationale.
- `money` does not validate TLS certificates beyond the Go standard library defaults.
