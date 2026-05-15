# Plaid Link Hardening Follow-up Plan

Date: 2026-05-13

## Goal

Improve the existing browser Plaid Link flow after Plaid Dashboard login is stable, without coupling this work to Dashboard OAuth.

## Scope

- Add Plaid Link `onExit` callback handling.
- Return structured cancellation without exchanging a public token.
- Return Plaid Link errors with request ID, link session ID, error type, and error code when Plaid provides them.
- Add host and origin validation to the short-lived callback helper.
- Keep state validation.
- Keep helper routes limited to `/` and `/callback`.
- Keep shutdown after callback or timeout.
- Add Plaid Link product consent ergonomics:
  - `additional_consented_products`
  - `required_if_supported_products`
  - `optional_products`
- Keep `products` as the explicit initial product set.
- Validate Plaid product names before calling Plaid; unsupported names must not reach the API.
- Do not expand local stable read contracts just because a product is requested. New data appears only after the local model and command contracts support it.

## Out of Scope

- Plaid Dashboard OAuth.
- Plaid Dashboard teams or key fetch.
- Plaid Sandbox direct linking.
- New local read contracts for products that are not already modeled.

## Implementation Checklist

- [x] Re-read `docs/ARCHITECTURE.md` Plaid Link helper rules.
- [x] Update `internal/linking/plaid_helper.go` to handle `onExit`.
- [x] Add structured callback payloads for success, cancel, and Link error.
- [x] Add host/origin validation without allowing cross-origin financial data access.
- [x] Add provider/config fields for additional consent product options.
- [x] Add CLI flags only where they match existing command style.
- [x] Extend `BuildPlaidLinkTokenCreateRequest` for the product consent options.
- [x] Add tests for success, cancel, Link error, wrong state, wrong origin, duplicate callback, and timeout.
- [x] Add product validation tests.
- [x] Update docs/contracts only for exposed command or JSON behavior changes.
