package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"

	"github.com/whilesmartgo/apicontract"
	"golang.org/x/tools/go/packages"
)

const contractPath = "github.com/whilesmartgo/apicontract"

// contractParams returns the query parameters of every struct the handler
// passes to apicontract.BindQuery or apicontract.BindValues, described by the
// same rules the parser applies.
func contractParams(api *packages.Package, fn *ast.FuncDecl) ([]parameter, error) {
	var params []parameter
	var failure error
	seen := map[string]bool{}
	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 || failure != nil {
			return true
		}
		if !isContractBind(api, call.Fun) {
			return true
		}
		target, ok := api.TypesInfo.TypeOf(call.Args[1]).(*types.Pointer)
		if !ok {
			return true
		}
		structure, ok := target.Elem().Underlying().(*types.Struct)
		if !ok {
			return true
		}
		described, err := apicontract.OpenAPIParametersOf(contractFields(structure, 0))
		if err != nil {
			failure = fmt.Errorf("%s: %w", target.Elem(), err)
			return false
		}
		for _, item := range described {
			name, _ := item["name"].(string)
			if seen[name] {
				continue
			}
			seen[name] = true
			description, _ := item["description"].(string)
			required, _ := item["required"].(bool)
			deprecated, _ := item["deprecated"].(bool)
			params = append(params, parameter{Name: name, In: "query", Description: description, Required: required, Deprecated: deprecated, Schema: item["schema"]})
		}
		return true
	})
	return params, failure
}

func isContractBind(api *packages.Package, fun ast.Expr) bool {
	var ident *ast.Ident
	switch expr := fun.(type) {
	case *ast.SelectorExpr:
		ident = expr.Sel
	case *ast.Ident:
		ident = expr
	default:
		return false
	}
	object, ok := api.TypesInfo.Uses[ident].(*types.Func)
	if !ok || object.Pkg() == nil || object.Pkg().Path() != contractPath {
		return false
	}
	return object.Name() == "BindQuery" || object.Name() == "BindValues"
}

func contractFields(structure *types.Struct, depth int) []apicontract.Field {
	fields := make([]apicontract.Field, structure.NumFields())
	for i := range fields {
		variable := structure.Field(i)
		field := apicontract.Field{
			Name:      variable.Name(),
			Exported:  variable.Exported(),
			Anonymous: variable.Embedded(),
			Tag:       reflect.StructTag(structure.Tag(i)),
			Type:      contractType(variable.Type()),
		}
		if field.Type == apicontract.FieldStruct && depth < 8 {
			field.Range = isNamed(variable.Type(), contractPath, "Range")
			field.Fields = contractFields(variable.Type().Underlying().(*types.Struct), depth+1)
		}
		fields[i] = field
	}
	return fields
}

func contractType(t types.Type) apicontract.FieldType {
	switch {
	case isNamed(t, "time", "Time"):
		return apicontract.FieldTime
	}
	if pointer, ok := t.(*types.Pointer); ok && isNamed(pointer.Elem(), "time", "Location") {
		return apicontract.FieldLocation
	}
	if slice, ok := t.Underlying().(*types.Slice); ok {
		if isNamed(slice.Elem(), contractPath, "SortKey") {
			return apicontract.FieldSort
		}
		if basic, ok := slice.Elem().(*types.Basic); ok && basic.Kind() == types.String {
			return apicontract.FieldStrings
		}
		return apicontract.FieldOther
	}
	switch underlying := t.Underlying().(type) {
	case *types.Struct:
		return apicontract.FieldStruct
	case *types.Basic:
		switch {
		case underlying.Kind() == types.String:
			return apicontract.FieldString
		case underlying.Kind() == types.Bool:
			return apicontract.FieldBool
		case underlying.Info()&types.IsInteger != 0 && underlying.Info()&types.IsUnsigned == 0:
			return apicontract.FieldInt
		}
	}
	return apicontract.FieldOther
}

func isNamed(t types.Type, path, name string) bool {
	named, ok := t.(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == path && named.Obj().Name() == name
}
