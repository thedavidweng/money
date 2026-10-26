# Plaid v49 SDK

## Status

Accepted.

## Context

The stable Plaid SDK is v49.0.0. Its module replaces the v40 module used by the provider and introduces a different client and model API.

## Decision

Upgrade to v49.0.0 using its official `legacy` package, as prescribed by [Plaid's migration guide](https://github.com/plaid/plaid-go/blob/v49.0.0/MIGRATION.md#switch-your-imports-to-the-legacy-package). Keep existing provider calls, request validation, mappings and error handling unchanged. Verify the existing provider and CLI workflows before merging.

## Consequences

The project uses the current stable SDK without rewriting unrelated financial-data behavior. The package name reflects Plaid's supported compatibility API, not a local fallback. Moving to the new client API is a separate behavioral migration, not required to consume v49.
