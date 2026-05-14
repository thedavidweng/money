# Plaid Sandbox Link Follow-up Plan

Date: 2026-05-13

## Goal

Add a direct Plaid Sandbox Item creation command after Dashboard login and the current browser Link flow are stable.

This is separate from Dashboard OAuth. It uses Plaid's public Sandbox API surface and exists to make local development and verification easier.

## Command Surface

```text
money plaid sandbox link
money plaid sandbox link --products transactions,liabilities --institution-id ins_56
```

## Behavior

- Requires Plaid API credentials in the selected `money` config.
- Only works when Plaid environment is `sandbox`.
- Calls Plaid Sandbox public-token creation.
- Reuses the same public-token exchange and Provider Item persistence path as browser Plaid Link.
- Stores the resulting Provider Item in the encrypted `money` store.
- Does not run `money sync` automatically.
- First version rejects `balance`; do not implement Plaid CLI's `balance -> transactions` mapping until current Plaid docs/API behavior is re-verified and the product decision is explicit.

## Out of Scope

- Plaid Dashboard OAuth.
- Browser Plaid Link hardening.
- Plaid CLI's `SANDBOX_PRODUCT_MAPPING` diagnostic behavior.
- Multi-environment Plaid credential vaulting.

## Implementation Checklist

- [x] Re-check current Plaid Sandbox public-token docs before coding.
- [x] Add Sandbox public-token create client method.
- [x] Add `money plaid sandbox link` under the Plaid-specific command namespace.
- [x] Enforce `environment == sandbox`.
- [x] Reject `balance` explicitly with a validation error.
- [x] Reuse public-token exchange and Provider Item persistence.
- [x] Add tests for sandbox-only enforcement, product validation, balance rejection, public-token exchange, and persistence.
- [x] Update README / docs only if the command is included in the shipped command surface.
