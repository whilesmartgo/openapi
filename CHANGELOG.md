# Changelog

## 0.2.0 (unreleased)

- Describe query structs passed to `apicontract.BindQuery` or `apicontract.BindValues`, including types, formats, bounds and deprecation.
- Use the parser's request body schema for structs passed to `apicontract.BindJSON` or `apicontract.BindJSONBytes`.
- Document `application/problem+json` responses for calls to `apicontract` parsers.
- Describe responses written with `apicontract.WriteJSON`, `WriteList` and `WriteError`, plus `WriteProblem` passed a `NewProblem` with a constant error status. Omit bodies for `204` and `205` responses.
- Follow type aliases and defined pointer types as `apicontract` does at run time. Read calls with explicit type arguments and closures called on the spot, including parenthesized closures. Skip uncalled closures.
- Record the declared API contract version in `info.x-whilesmart-contract` with `-contract`.

## 0.1.1 (2026-11-05)

- Resolve handlers in routes registered from other packages, so their request bodies, responses and schemas are documented.

## 0.1.0 (2026-08-30)

- Generate OpenAPI 3.1 documents from registered Gin routes.
- Derive parameters and schemas from Go handler types.
- Add operation details and field descriptions through typed metadata.
- Check generated documents for drift in CI.
