# OpenAPI for Go

Keep your OpenAPI document in step with your Go service.

API documents drift when routes, request types, and documentation are maintained in different places. This package reads your registered Gin routes and the Go types used by their handlers. It writes an OpenAPI 3.1 document that CI can check.

Use it when you want accurate OpenAPI without annotation blocks or a second list of routes.

## What it reads

The generator finds:

- registered routes and HTTP methods
- path and query parameters
- request bodies bound by handlers
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
