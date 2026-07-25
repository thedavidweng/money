# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [0.4.0](https://github.com/thedavidweng/money/compare/v0.3.1...v0.4.0) (2026-07-25)


### Features

* align with CLI fleet standard ([c1f355e](https://github.com/thedavidweng/money/commit/c1f355e97fa88064663030c77097e3e6ef893278))
* unify JSON envelope with the fleet core schema ([554a84e](https://github.com/thedavidweng/money/commit/554a84e9af6d8811a2cbfd2ad02eef661f092be2))


### Bug Fixes

* add force_push and fetch-depth: 0 ([958a29e](https://github.com/thedavidweng/money/commit/958a29e009f76e4685a7a9caf8edab5f36d17d29))
* add force_push and fetch-depth: 0 ([8733c25](https://github.com/thedavidweng/money/commit/8733c2587a8eea604cb68e24bacba805895231e1))
* address Go production audit findings ([cd52f55](https://github.com/thedavidweng/money/commit/cd52f55a2c9490a018f329105d0f8e14ab8ba952))
* address PR review — golangci-lint v2 config and benchmark reset ([15799f3](https://github.com/thedavidweng/money/commit/15799f37114f708e796a0b81edb35d39d1b20a52))
* correct mirror action SHA ([e282135](https://github.com/thedavidweng/money/commit/e28213528e17dae471d92c4b0a9b28b4fcbfcba1))
* correct mirror action SHA ([a82ea2b](https://github.com/thedavidweng/money/commit/a82ea2b59823f119b4aeaaa1ed4d7ac5c6905a51))
* **deps:** clear all security advisories and adopt tablewriter v1 ([6205c8b](https://github.com/thedavidweng/money/commit/6205c8b801548b6db5682140d56ff872ca0bbe76))
* gofmt test files and normalize configSkeleton paths for Windows ([a7b7f3a](https://github.com/thedavidweng/money/commit/a7b7f3a514ebc1f607c1094683654b653e20654e))
* mock openBrowser in TestFeedbackHumanMode ([e71ec6e](https://github.com/thedavidweng/money/commit/e71ec6ec9f2e0775626e64e5daef0a67c85f9223))
* move codecov ignore to top level ([9fcff5a](https://github.com/thedavidweng/money/commit/9fcff5adb598434bbef0728c84b7e287adfda8c6))
* pin action SHA, remove test.txt, add permissions ([f604b46](https://github.com/thedavidweng/money/commit/f604b463b25f6c37ad383832d011bcb1ebf4ef87))
* remove invalid generate_completions_from_executable from cask hooks ([464bd06](https://github.com/thedavidweng/money/commit/464bd067ae03b94db52a33cc8d260b1474f03dc8))
* repair release-please workflow YAML ([c60bc35](https://github.com/thedavidweng/money/commit/c60bc35f8ce7c30cfb85fe0ba310ab6b70049436))


### Documentation

* add Go Report Card badge ([197948f](https://github.com/thedavidweng/money/commit/197948f35778794e5b372602e58bf75c72db7085))
* remove completed plans and stale agent docs ([48a9b1d](https://github.com/thedavidweng/money/commit/48a9b1d6eabaa8f51e111989d49c822172f7cf36))
* remove old Astro website, point to unified VitePress site ([1e17ca0](https://github.com/thedavidweng/money/commit/1e17ca08c57d9cc60a9345f97a6b5be22b1fc051))
* standardize README badges ([01d6d0f](https://github.com/thedavidweng/money/commit/01d6d0f30680af44386b4a5b074bc7e6f260c3b5))

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
- ARCHITECTURE.md future features (cleanup, accounts update, --pretty) marked as [Planned].
- Website license references corrected from MIT to Apache 2.0.
- README monarch import description corrected (CSV → data sources).
- Removed dead staticProvider code from providers/registry.go.

### Changed
- GoReleaser config: added auto-changelog, GPG signing, SBOM generation, macOS universal binaries.
- Release workflow: added Syft (SBOM) and GPG key import steps.

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
