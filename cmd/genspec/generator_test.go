package main

import (
	"path/filepath"
	"testing"
)

func TestBuildReadsDescriptionsFromTypedRoutes(t *testing.T) {
	root := filepath.Join("testdata", "service")
	document, err := build(root, "Fixture API", true)
	if err != nil {
		t.Fatal(err)
	}

	operation := document.Paths["/events"]["post"]
	if operation.Summary != "Create an event" || operation.Description != "Creates an event for the selected owner." || !operation.Deprecated {
		t.Fatalf("unexpected operation metadata: %#v", operation)
	}
	if len(operation.Tags) != 1 || operation.Tags[0] != "calendar" {
		t.Fatalf("unexpected operation tags: %v", operation.Tags)
	}

	input := document.Components.Schemas["server.eventInput"]
	title := input.Properties["title"]
	if title.Description != "Event title" || title.Example != "Planning session" {
		t.Fatalf("unexpected field metadata: %#v", title)
	}
}

func TestBuildResolvesHandlersRegisteredInAnotherPackage(t *testing.T) {
	document, err := build(filepath.Join("testdata", "service"), "Fixture API", true)
	if err != nil {
		t.Fatal(err)
	}

	operation := document.Paths["/notes"]["post"]
	if operation == nil || operation.RequestBody == nil {
		t.Fatalf("expected a request body for POST /notes: %#v", operation)
	}
	if _, ok := document.Components.Schemas["notes.noteInput"]; !ok {
		t.Fatalf("expected the notes.noteInput schema, got %v", document.Components.Schemas)
	}
}
