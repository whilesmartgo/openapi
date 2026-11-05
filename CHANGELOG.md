# Changelog

## 0.1.1 (2026-11-05)

- Resolve handlers in routes registered from other packages, so their request bodies, responses and schemas are documented.

## 0.1.0 (2026-08-30)

- Generate OpenAPI 3.1 documents from registered Gin routes.
- Derive parameters and schemas from Go handler types.
- Add operation details and field descriptions through typed metadata.
- Check generated documents for drift in CI.
