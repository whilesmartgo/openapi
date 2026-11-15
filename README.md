# OpenAPI for Go

Keep your OpenAPI document in step with your Go service.

API documents drift when routes, request types, and documentation are maintained in different places. This package reads your registered Gin routes and the Go types used by their handlers. It writes an OpenAPI 3.1 document that CI can check.

Use it when you want accurate OpenAPI without annotation blocks or a second list of routes.

## What it reads

The generator finds:

- registered routes and HTTP methods
- path and query parameters, including typed query structs read with [`apicontract.BindQuery`](https://github.com/whilesmartgo/apicontract)
- request bodies bound by handlers, including structs read with `apicontract.BindJSON`
- problem responses for calls to `apicontract` parsers
- typed JSON responses
- required fields and enum values from validation tags
- operation and field descriptions you choose to add

## Add descriptions

Wrap a handler with `Describe`. The wrapper returns the same handler, so it does not change request handling.

```go
router.POST("/events", openapi.Describe(events.Create, openapi.Operation{
	Summary:     "Create an event",
	Description: "Creates an event for the selected owner.",
	Tags:        []string{"calendar"},
}))
```

Describe request fields with tags:

```go
type EventInput struct {
	Title string `json:"title" binding:"required" doc:"Event title" example:"Planning session"`
}
```

Descriptions are optional. Routes and schemas are still generated when no prose is supplied.

## API contract structs

The generator reads query structs passed to `apicontract.BindQuery` or `apicontract.BindValues`. It uses the parser's rules for parameter names, types, formats, bounds and deprecation.

For structs passed to `apicontract.BindJSON` or `apicontract.BindJSONBytes`, it uses the parser's request body schema. This includes required fields, bounds, nullable optional pointers and maps, nested objects and `additionalProperties: false` for struct objects.

It also documents `application/problem+json` responses for handler calls to:

- `BindQuery` or `BindValues`: `400` and `422`
- `BindJSONBytes`: `400`, `413` and `422`
- `BindJSON`: `400`, `413`, `415` and `422`
- `apicontract.IdempotencyKey`: `400`

Responses written with `apicontract.WriteJSON` at a constant success status, or with `apicontract.WriteList`, get schemas from their types. Responses with status `204` or `205` have no body. `WriteError` and `WriteProblem` passed a `NewProblem` add problem responses for constant error statuses.

The generator reads calls made directly in a handler method, including calls with explicit type arguments and closures called on the spot, with or without parentheses. It skips uncalled closures and does not follow calls through variables or helper functions. Calls hidden in wrappers therefore add no body or responses. A body struct with an unresolved type parameter stops generation with an error.

Pass `-contract 0.2` to write `0.2` to `info.x-whilesmart-contract`. This records the declared contract version; it does not check compliance.

## Install

Add the generator as a pinned Go tool:

```text
go get -tool github.com/whilesmartgo/openapi/cmd/genspec@latest
```

Add the package when you use `Describe`:

```text
go get github.com/whilesmartgo/openapi@latest
```

## Generate

Run the tool from your service module:

```text
go tool genspec -root . -title "Events API" -o cmd/server/openapi.json
```

The service can embed and serve the generated file:

```go
//go:embed openapi.json
var specification []byte
```

## Stop drift in CI

Generate the document and check that the working tree did not change:

```text
go tool genspec -root . -title "Events API" -o cmd/server/openapi.json
git diff --exit-code -- cmd/server/openapi.json
```

A route or type change now requires its OpenAPI document to change in the same commit.
