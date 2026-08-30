package main

import (
	"go/token"
	"go/types"
	"testing"
)

func TestUUIDSchema(t *testing.T) {
	object := types.NewTypeName(token.NoPos, types.NewPackage("github.com/google/uuid", "uuid"), "UUID", nil)
	typeOfUUID := types.NewNamed(object, types.NewArray(types.Typ[types.Byte], 16), nil)
	schemas := &schemaSet{byName: map[string]*schema{}, seen: map[string]bool{}}

	generated := schemas.add(typeOfUUID)
	if generated == nil || generated.Type != "string" || generated.Format != "uuid" {
		t.Fatalf("unexpected UUID schema: %#v", generated)
	}
}
