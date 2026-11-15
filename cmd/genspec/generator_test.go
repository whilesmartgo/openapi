package main

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/whilesmartgo/apicontract/lint"
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

func TestBuildDescribesContractBodiesAndProblems(t *testing.T) {
	document, err := build(filepath.Join("testdata", "service"), "Fixture API", true)
	if err != nil {
		t.Fatal(err)
	}
	operation := document.Paths["/events/{id}"]["patch"]
	if operation == nil || operation.RequestBody == nil {
		t.Fatalf("PATCH /events/{id} must have a request body: %#v", operation)
	}
	data, _ := json.Marshal(operation.RequestBody)
	for _, want := range []string{
		`"additionalProperties":false`,
		`"title":{"maxLength":200,"type":["string","null"]}`,
		`"day":{"format":"date","type":"string"}`,
		`"attendees":{"items":{"additionalProperties":false,"properties":{"name":{"type":"string"}},"required":["name"],"type":"object"},"maxItems":10,"type":"array"}`,
		`"metadata":{"type":["object","null"]}`,
	} {
		if !strings.Contains(string(data), want) {
			t.Errorf("request body %s lacks %s", data, want)
		}
	}
	for _, status := range []string{"400", "413", "415", "422"} {
		response, ok := operation.Responses[status]
		if !ok {
			t.Fatalf("missing %s response: %v", status, operation.Responses)
		}
		if _, ok := response.Content["application/problem+json"]; !ok {
			t.Fatalf("%s response must be a problem: %#v", status, response)
		}
	}
	if _, ok := document.Paths["/events"]["get"].Responses["422"]; !ok {
		t.Fatal("GET /events binds a query and must declare 422")
	}
}

func TestContractRoutesPassTheContractLint(t *testing.T) {
	document, err := build(filepath.Join("testdata", "service"), "Fixture API", true)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(document)
	violations, err := lint.Check("fixture", data)
	if err != nil {
		t.Fatal(err)
	}
	for _, violation := range violations {
		if violation.Path == "/events/{id}" {
			t.Errorf("generated route breaks the contract: %s", violation)
		}
	}
}

func TestBuildFollowsTheContractWriters(t *testing.T) {
	document, err := build(filepath.Join("testdata", "service"), "Fixture API", true)
	if err != nil {
		t.Fatal(err)
	}
	create := document.Paths["/tasks"]["post"]
	if create == nil || create.RequestBody == nil {
		t.Fatalf("POST /tasks must have a body: %#v", create)
	}
	body, _ := json.Marshal(create.RequestBody)
	for _, want := range []string{
		`"due_at":{"format":"date-time","type":"string"}`,
		`"count":{"maximum":5,"type":["integer","null"]}`,
		`"level":{"maximum":127,"minimum":-128,"type":"integer"}`,
	} {
		if !strings.Contains(string(body), want) {
			t.Errorf("body %s lacks %s", body, want)
		}
	}
	if strings.Contains(string(body), "secret") {
		t.Errorf("a closure that never runs must not supply the body: %s", body)
	}
	for _, status := range []string{"201", "400", "409", "413", "422"} {
		if _, ok := create.Responses[status]; !ok {
			t.Errorf("POST /tasks lacks %s: %v", status, create.Responses)
		}
	}
	for _, status := range []string{"200", "415"} {
		if _, ok := create.Responses[status]; ok {
			t.Errorf("POST /tasks must not declare %s", status)
		}
	}
	list, _ := json.Marshal(document.Paths["/tasks"]["get"].Responses["200"])
	if !strings.Contains(string(list), `"required":["items"]`) || !strings.Contains(string(list), `"next_cursor":{"type":"string"}`) {
		t.Errorf("WriteList must describe the list envelope: %s", list)
	}
}

func TestBuildReadsEveryContractWriterForm(t *testing.T) {
	document, err := build(filepath.Join("testdata", "service"), "Fixture API", true)
	if err != nil {
		t.Fatal(err)
	}
	statuses := func(path, method string) []string {
		var out []string
		for status := range document.Paths[path][method].Responses {
			out = append(out, status)
		}
		sort.Strings(out)
		return out
	}
	for _, check := range []struct {
		path, method string
		want         []string
	}{
		{"/tasks/{id}", "delete", []string{"204"}},
		{"/tasks/reset", "post", []string{"205"}},
		{"/tasks/{id}", "put", []string{"200", "400", "409", "413", "415", "422"}},
	} {
		if got := statuses(check.path, check.method); !reflect.DeepEqual(got, check.want) {
			t.Errorf("%s %s: got %v, want %v", check.method, check.path, got, check.want)
		}
	}
	if content := document.Paths["/tasks/reset"]["post"].Responses["205"].Content; content != nil {
		t.Errorf("205 must not have content: %v", content)
	}
	if document.Paths["/tasks/{id}"]["put"].RequestBody == nil {
		t.Error("a closure called in parentheses must supply the body")
	}
	list, _ := json.Marshal(document.Paths["/tasks"]["get"].Responses["200"])
	if !strings.Contains(string(list), `"required":["items"]`) {
		t.Errorf("WriteList with explicit type arguments must describe the envelope: %s", list)
	}
}
