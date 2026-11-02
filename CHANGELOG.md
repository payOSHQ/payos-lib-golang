# Changelog

## 2.0.2 (2026-10-30)

### Bug Fixes

- **webhooks:** stricter validation of webhook data before signature verification; no panic on a non-string signature ([GHSA-ghjh-8q98-fg8g](https://github.com/payOSHQ/payos-lib-golang/security/advisories/GHSA-ghjh-8q98-fg8g))
- **crypto:** constant-time signature comparison; GenerateUUID uses crypto/rand
- **client:** percent-encode path segments (ids, order codes, invoice ids, payout ids) and reject empty, `.`/`..`, negative or unsafe integer values; strip credentials (`x-api-key`, `x-client-id`, `x-partner-code`, `Authorization`, `Cookie`, `Proxy-Authorization`) for the rest of a redirect chain once it leaves the API origin, including cookie-jar cookies (max 5 redirects, not retried; also for a custom `HTTPClient`); validate body-signed response data before signature verification

## 2.0.0 (2025-11-07)

This release introduces a major refactor with structured client initialization, context support, resource-based methods, and new payout features. For full migration guide see [MIGRATION.md](./MIGRATION.md).

### Features

- **api:** add `/v2/payment-requests/invoices`
- **api:** add `/v1/payouts`
- **api:** add `/v1/payouts-account`
- **client:** add `NewPayOS()` constructor with `PayOSOptions` for better configuration
- **client:** add context support for all API methods
- **client:** add resource-based methods under `PaymentRequests`, `Webhooks`, `Payouts`, and `PayoutsAccount`
- **client:** add `client.Crypto` package for signature calculation
- **client:** add custom HTTP client support via `PayOSOptions.HTTPClient`
- **client:** add request timeout configuration via `PayOSOptions.Timeout`
- **client:** add retry support with configurable `MaxRetries`
- **client:** add debug logging via `PayOSOptions.DebugLogger`
- **client:** add pagination support with auto-paging iterators
- **client:** add middleware support for request/response interception
- **client:** add file download support for invoice operations
- **client:** add better error handling with `APIError`, `WebhookError`, and `InvalidSignatureError`
- **types:** rename types for better clarity (e.g., `CheckoutRequestType` => `CreatePaymentLinkRequest`)

### Documentation

- **readme:** update readme with comprehensive usage examples
- **migration:** add migration guide

## 1.0.7 (2024-09-27)

### Bug fixes

- **client:** add `expiredAt` to `CheckoutResponseDataType`

## 1.0.6 (2024-07-03)

### Features

- **client:** add partnerCode in create payment link header

## 1.0.5 (2024-06-20)

### Bug fixes

- **client:** fix json string in signature

## 1.0.4 (2024-02-17)

### Bug fixes

- **client:** change type of `orderCode` to `int64`

## 1.0.3 (2024-01-12)

### Bug fixes

- **client:** convert numbers to strings in signature

## 1.0.2 (2024-01-09)

### Bug fixes

- **client:** add `Currency` to `CheckoutResponseDataType` and `WebhookDataType`

## 1.0.1 (2024-01-02)

### Bug fixes

- **client:** change PayOS base URL

## 1.0.0 (2023-12-28)

- Initial version
