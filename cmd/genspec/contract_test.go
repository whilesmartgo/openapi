package main

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/whilesmartgo/apicontract"
)

func TestDescribeRejectsUnresolvedTypeParameters(t *testing.T) {
	parameter := types.NewTypeParam(types.NewTypeName(token.NoPos, nil, "T", nil), types.NewInterfaceType(nil, nil))
	for _, input := range []types.Type{
		parameter,
		types.NewMap(types.Typ[types.String], parameter),
		types.NewSlice(parameter),
	} {
		if got := describe(input, map[types.Type]bool{}); got.Type == apicontract.FieldObject || got.Type == apicontract.FieldAny {
			t.Errorf("%s described as %v", input, got.Type)
		}
	}
}
