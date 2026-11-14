package main

import (
	"path/filepath"
	"strings"
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

func TestBuildDescribesContractQueryStructs(t *testing.T) {
	document, err := build(filepath.Join("testdata", "service"), "Fixture API", true)
	if err != nil {
		t.Fatal(err)
	}
	operation := document.Paths["/events"]["get"]
	if operation == nil {
		t.Fatal("GET /events is missing")
	}
	byName := map[string]parameter{}
	var names []string
	for _, p := range operation.Parameters {
		byName[p.Name] = p
		names = append(names, p.Name)
	}
	want := []string{"calendar_id", "limit", "cursor", "updated_from", "updated_to", "sort", "start"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("parameters %v, want %v", names, want)
	}
	limit := byName["limit"].Schema.(map[string]any)
	if limit["type"] != "integer" || limit["maximum"] != 200 {
		t.Fatalf("limit must come from the contract struct, not the c.Query call: %v", limit)
	}
	to := byName["updated_to"].Schema.(map[string]any)
	if _, ok := to["anyOf"]; !ok {
		t.Fatalf("updated_to must declare its date formats: %v", to)
	}
	if !byName["start"].Deprecated {
		t.Fatal("start must be deprecated")
	}
}
