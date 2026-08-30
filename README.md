# OpenAPI for Go

Generate OpenAPI 3.1 documents from registered Gin routes and the Go types used by their handlers.

```go
router.POST("/events", openapi.Describe(events.Create, openapi.Operation{
    Summary:     "Create an event",
    Description: "Creates an event for the selected owner.",
}))
```

Request field descriptions and examples use struct tags:

```go
type EventInput struct {
    Title string `json:"title" binding:"required" doc:"Event title" example:"Planning session"`
}
```

Generate a document from a service module:

```text
go get -tool github.com/whilesmartgo/openapi/cmd/genspec@latest
go tool genspec -root . -title "Events API" -o cmd/server/openapi.json
```
