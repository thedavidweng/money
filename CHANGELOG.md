# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.3.0] - 2026-06-03

### Added
- SECURITY.md with vulnerability reporting policy and known limitations.
- CHANGELOG.md to track changes across releases.
- SCHEMA.md updated with investment (securities, holdings), liability, budget, and rules table DDL.
- CONTRIBUTING.md expanded with Go version prerequisite, race-detected testing, and release workflow instructions.
- `money doctor` now includes Links and Sync diagnostic sections.
- `money doctor --fix` can remove errored provider items and mark stuck sync runs as interrupted.
- Structured logging (slog) in doctor diagnostics when stderr is a TTY.
- `LatestSyncRuns` and `MarkStuckSyncRunsInterrupted` store methods.

### Fixed
- GETTING_STARTED.md license reference corrected from MIT to Apache 2.0.
- Go version requirement updated to 1.26 across go.mod, README, and documentation.
- README command list updated with cashflow, net-worth, budgets, rules, import, and tx alias commands.
- README architecture section now lists all internal packages.
- README removed unimplemented MX/Finicity provider references.
- Release workflow now runs `go test ./... -race` before GoReleaser.
- Removed unused cosign installer from release workflow.
- ARCHITECTURE.md future features (cleanup, accounts update, --pretty) marked as [Planned].

## [0.2.0] - 2026-05-17

### Fixed
- GoReleaser post-install xattr hook for macOS quarantine removal.

## [0.1.0] - 2026-05-10

### Added
- Initial release with 23 CLI commands.
- Encrypted SQLite store with Adiantum VFS.
- Plaid and Bridge provider adapters.
- Monarch CSV import source.
- `money setup` guided onboarding with interactive Plaid Dashboard login.
- `money doctor` configuration diagnostics.
- Investments, liabilities, budgets, and rules support.
- E2E test suite with command coverage reporting.
- CI pipeline (Go vet, race-detected tests, multi-platform build).
- GoReleaser-based release automation with Homebrew cask.
- Astro-based project landing page.

[Unreleased]: https://github.com/thedavidweng/money/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/thedavidweng/money/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/thedavidweng/money/releases/tag/v0.1.0
