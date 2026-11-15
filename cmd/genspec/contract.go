package main

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"reflect"
	"strconv"

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
	inspectDirect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 || failure != nil {
			return true
		}
		if name := contractCall(api, call.Fun); name != "BindQuery" && name != "BindValues" {
			return true
		}
		target, ok := pointerTo(api.TypesInfo.TypeOf(call.Args[1]))
		if !ok {
			return true
		}
		structure, ok := target.Underlying().(*types.Struct)
		if !ok {
			return true
		}
		described, err := apicontract.OpenAPIParametersOf(contractFields(structure, map[types.Type]bool{target: true}))
		if err != nil {
			failure = fmt.Errorf("%s: %w", target, err)
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

// contractCall returns the name of the apicontract function fun refers to, or
// an empty string.
func contractCall(api *packages.Package, fun ast.Expr) string {
	var ident *ast.Ident
	fun = ast.Unparen(fun)
	switch expr := fun.(type) {
	case *ast.IndexExpr:
		fun = expr.X
	case *ast.IndexListExpr:
		fun = expr.X
	}
	switch expr := ast.Unparen(fun).(type) {
	case *ast.SelectorExpr:
		ident = expr.Sel
	case *ast.Ident:
		ident = expr
	default:
		return ""
	}
	object, ok := api.TypesInfo.Uses[ident].(*types.Func)
	if !ok || object.Pkg() == nil || object.Pkg().Path() != contractPath {
		return ""
	}
	return object.Name()
}

// contractBody describes the struct the handler passes to apicontract.BindJSON
// or BindJSONBytes as an OpenAPI request body.
// When it binds more than one struct, the body schema is a oneOf of them.
func contractBody(api *packages.Package, fn *ast.FuncDecl) (any, error) {
	var bodies []map[string]any
	seen := map[types.Type]bool{}
	var failure error
	inspectDirect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || failure != nil || len(call.Args) < 2 {
			return failure == nil
		}
		if name := contractCall(api, call.Fun); name != "BindJSON" && name != "BindJSONBytes" {
			return true
		}
		target, ok := pointerTo(api.TypesInfo.TypeOf(call.Args[1]))
		if !ok {
			return true
		}
		structure, ok := target.Underlying().(*types.Struct)
		if !ok || seen[target] {
			return true
		}
		seen[target] = true
		described, err := apicontract.OpenAPIRequestBodyOf(contractFields(structure, map[types.Type]bool{target: true}))
		if err != nil {
			failure = fmt.Errorf("%s: %w", target, err)
			return false
		}
		bodies = append(bodies, described)
		return true
	})
	if failure != nil || len(bodies) == 0 {
		return nil, failure
	}
	if len(bodies) == 1 {
		return bodies[0], nil
	}
	var options []any
	for _, body := range bodies {
		options = append(options, body["content"].(map[string]any)["application/json"].(map[string]any)["schema"])
	}
	return map[string]any{
		"required": true,
		"content":  map[string]any{"application/json": map[string]any{"schema": map[string]any{"oneOf": options}}},
	}, nil
}

// contractErrors returns the problem responses a handler can write through
// apicontract: 400 and 422 when it binds a query or body, 413 when it reads a
// body, 415 when it reads one from a request, 400 when it reads an
// idempotency key, and the status of each WriteError, or WriteProblem of a
// NewProblem, with a constant status.
func contractErrors(api *packages.Package, fn *ast.FuncDecl) map[string]response {
	statuses := map[int]bool{}
	inspectDirect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch contractCall(api, call.Fun) {
		case "BindQuery", "BindValues":
			statuses[400], statuses[422] = true, true
		case "BindJSONBytes":
			statuses[400], statuses[413], statuses[422] = true, true, true
		case "BindJSON":
			statuses[400], statuses[413], statuses[415], statuses[422] = true, true, true, true
		case "IdempotencyKey":
			statuses[400] = true
		case "WriteError":
			if status, ok := constantStatus(api, call, 1); ok && status >= 400 {
				statuses[status] = true
			}
		case "WriteProblem":
			if len(call.Args) < 2 {
				break
			}
			made, ok := ast.Unparen(call.Args[1]).(*ast.CallExpr)
			if !ok || contractCall(api, made.Fun) != "NewProblem" {
				break
			}
			if status, ok := constantStatus(api, made, 0); ok && status >= 400 {
				statuses[status] = true
			}
		}
		return true
	})
	responses := map[string]response{}
	for status := range statuses {
		described := apicontract.OpenAPIProblemResponse(status)
		content := described["content"].(map[string]any)[apicontract.ProblemContentType].(map[string]any)
		responses[strconv.Itoa(status)] = response{
			Description: described["description"].(string),
			Content:     map[string]mediaType{apicontract.ProblemContentType: {Schema: content["schema"]}},
		}
	}
	return responses
}

func contractFields(structure *types.Struct, seen map[types.Type]bool) []apicontract.Field {
	fields := make([]apicontract.Field, structure.NumFields())
	for i := range fields {
		variable := structure.Field(i)
		field := describe(variable.Type(), seen)
		field.Name, field.Exported, field.Anonymous = variable.Name(), variable.Exported(), variable.Embedded()
		field.Tag = reflect.StructTag(structure.Tag(i))
		fields[i] = field
	}
	return fields
}

// describe mirrors how apicontract classifies a reflect.Type, so a struct read
// from source is described exactly as the parser reads it at run time.
func describe(t types.Type, seen map[types.Type]bool) apicontract.Field {
	t = types.Unalias(t)
	if _, ok := t.(*types.TypeParam); ok {
		return apicontract.Field{Type: apicontract.FieldOther}
	}
	if pointer, ok := t.Underlying().(*types.Pointer); ok {
		if _, named := t.(*types.Named); !named && isNamed(pointer.Elem(), "time", "Location") {
			return apicontract.Field{Type: apicontract.FieldLocation}
		}
		field := describe(pointer.Elem(), seen)
		if field.Pointer || field.Type == apicontract.FieldLocation {
			return apicontract.Field{Type: apicontract.FieldOther}
		}
		field.Pointer = true
		return field
	}
	switch {
	case isNamed(t, "time", "Time"):
		return apicontract.Field{Type: apicontract.FieldTime}
	case isNamed(t, contractPath, "Date"):
		return apicontract.Field{Type: apicontract.FieldDate}
	case isNamed(t, "encoding/json", "RawMessage"):
		return apicontract.Field{Type: apicontract.FieldAny}
	}
	switch underlying := t.Underlying().(type) {
	case *types.Interface:
		if underlying.Empty() {
			return apicontract.Field{Type: apicontract.FieldAny}
		}
	case *types.Map:
		key, keyOK := underlying.Key().Underlying().(*types.Basic)
		_, generic := types.Unalias(underlying.Elem()).(*types.TypeParam)
		value, valueOK := underlying.Elem().Underlying().(*types.Interface)
		if keyOK && key.Kind() == types.String && valueOK && value.Empty() && !generic {
			return apicontract.Field{Type: apicontract.FieldObject}
		}
	case *types.Slice:
		if _, named := t.(*types.Named); !named {
			if isNamed(underlying.Elem(), contractPath, "SortKey") {
				return apicontract.Field{Type: apicontract.FieldSort}
			}
			if basic, ok := types.Unalias(underlying.Elem()).(*types.Basic); ok && basic.Kind() == types.String {
				return apicontract.Field{Type: apicontract.FieldStrings}
			}
		}
		elem := describe(underlying.Elem(), seen)
		return apicontract.Field{Type: apicontract.FieldList, Elem: &elem}
	case *types.Struct:
		if seen[t] {
			return apicontract.Field{Type: apicontract.FieldOther}
		}
		seen[t] = true
		defer delete(seen, t)
		return apicontract.Field{Type: apicontract.FieldStruct, Range: isNamed(t, contractPath, "Range"), Fields: contractFields(underlying, seen)}
	case *types.Basic:
		switch {
		case underlying.Kind() == types.String:
			return apicontract.Field{Type: apicontract.FieldString}
		case underlying.Kind() == types.Bool:
			return apicontract.Field{Type: apicontract.FieldBool}
		case underlying.Info()&types.IsInteger != 0 && underlying.Info()&types.IsUnsigned == 0:
			return apicontract.Field{Type: apicontract.FieldInt, Bits: bits(underlying)}
		case underlying.Kind() == types.Float32 || underlying.Kind() == types.Float64:
			return apicontract.Field{Type: apicontract.FieldNumber, Bits: bits(underlying)}
		}
	}
	return apicontract.Field{Type: apicontract.FieldOther}
}

// targetSizes holds the sizes of the platform the analyzed packages are built
// for, so int has the width reflection will report there.
var targetSizes types.Sizes = types.SizesFor("gc", "amd64")

func bits(basic *types.Basic) int {
	return int(targetSizes.Sizeof(basic) * 8)
}

// pointerTo returns the type a pointer argument points to, including defined
// pointer types, as reflection's pointer kind check accepts them.
func pointerTo(t types.Type) (types.Type, bool) {
	if t == nil {
		return nil, false
	}
	pointer, ok := types.Unalias(t).Underlying().(*types.Pointer)
	if !ok {
		return nil, false
	}
	return types.Unalias(pointer.Elem()), true
}

// inspectDirect walks the statements a handler runs: function literals are
// skipped unless they are called where they are written.
func inspectDirect(root ast.Node, visit func(ast.Node) bool) {
	invoked := map[*ast.FuncLit]bool{}
	ast.Inspect(root, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if literal, ok := ast.Unparen(call.Fun).(*ast.FuncLit); ok {
				invoked[literal] = true
			}
		}
		if literal, ok := n.(*ast.FuncLit); ok && !invoked[literal] {
			return false
		}
		return visit(n)
	})
}

func constantStatus(api *packages.Package, call *ast.CallExpr, index int) (int, bool) {
	if len(call.Args) <= index {
		return 0, false
	}
	value := api.TypesInfo.Types[call.Args[index]].Value
	if value == nil || value.Kind() != constant.Int {
		return 0, false
	}
	status, ok := constant.Int64Val(value)
	return int(status), ok && status >= 100 && status <= 599
}

// writtenResponse is a success response a handler writes with
// apicontract.WriteJSON or WriteList. For WriteList, value is the item type.
// value is nil for a payload with no static type, such as nil.
type writtenResponse struct {
	status string
	value  types.Type
	list   bool
}

// contractResponses returns every success response the handler writes, in
// source order, without repeats.
func contractResponses(api *packages.Package, fn *ast.FuncDecl) []writtenResponse {
	var found []writtenResponse
	add := func(next writtenResponse) {
		for _, existing := range found {
			if existing.status == next.status && existing.list == next.list && (existing.value == next.value || existing.value != nil && next.value != nil && types.Identical(existing.value, next.value)) {
				return
			}
		}
		found = append(found, next)
	}
	inspectDirect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch contractCall(api, call.Fun) {
		case "WriteJSON":
			code, ok := constantStatus(api, call, 1)
			if !ok || code >= 300 || len(call.Args) < 3 {
				return true
			}
			next := writtenResponse{status: strconv.Itoa(code)}
			if typed := api.TypesInfo.TypeOf(call.Args[2]); typed != nil && !isUntypedNil(typed) {
				next.value = typed
			}
			add(next)
		case "WriteList":
			if len(call.Args) < 2 {
				return true
			}
			if slice, ok := types.Unalias(api.TypesInfo.TypeOf(call.Args[1])).Underlying().(*types.Slice); ok {
				add(writtenResponse{status: "200", value: slice.Elem(), list: true})
			}
		}
		return true
	})
	return found
}

func isUntypedNil(t types.Type) bool {
	basic, ok := t.(*types.Basic)
	return ok && basic.Kind() == types.UntypedNil
}

func isNamed(t types.Type, path, name string) bool {
	named, ok := types.Unalias(t).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == path && named.Obj().Name() == name
}
