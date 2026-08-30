package main

import (
	"go/types"
	"reflect"
	"strings"
)

type openAPI struct {
	OpenAPI    string                           `json:"openapi"`
	Info       info                             `json:"info"`
	Servers    []server                         `json:"servers,omitempty"`
	Paths      map[string]map[string]*operation `json:"paths"`
	Components components                       `json:"components"`
	Security   []map[string][]string            `json:"security,omitempty"`
}

type server struct {
	URL string `json:"url"`
}

type info struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type components struct {
	Schemas         map[string]*schema `json:"schemas"`
	SecuritySchemes map[string]any     `json:"securitySchemes,omitempty"`
}

type operation struct {
	OperationID string              `json:"operationId"`
	Summary     string              `json:"summary,omitempty"`
	Description string              `json:"description,omitempty"`
	Tags        []string            `json:"tags,omitempty"`
	Deprecated  bool                `json:"deprecated,omitempty"`
	Parameters  []parameter         `json:"parameters,omitempty"`
	RequestBody *requestBody        `json:"requestBody,omitempty"`
	Responses   map[string]response `json:"responses"`
}

type parameter struct {
	Name     string  `json:"name"`
	In       string  `json:"in"`
	Required bool    `json:"required,omitempty"`
	Schema   *schema `json:"schema,omitempty"`
	Rest     bool    `json:"x-rest-of-path,omitempty"`
}

type requestBody struct {
	Required bool                 `json:"required,omitempty"`
	Content  map[string]mediaType `json:"content"`
}

type response struct {
	Description string               `json:"description"`
	Content     map[string]mediaType `json:"content,omitempty"`
}

type mediaType struct {
	Schema *schema `json:"schema,omitempty"`
}

type schema struct {
	Ref                  string             `json:"$ref,omitempty"`
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Items                *schema            `json:"items,omitempty"`
	Properties           map[string]*schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Description          string             `json:"description,omitempty"`
	Example              string             `json:"example,omitempty"`
	AdditionalProperties *schema            `json:"additionalProperties,omitempty"`
	Enum                 []string           `json:"enum,omitempty"`
}

type schemaSet struct {
	byName map[string]*schema
	seen   map[string]bool
}

func (s *schemaSet) add(t types.Type) *schema {
	built := s.build(t, 0)
	if built == nil {
		return nil
	}
	return built
}

func (s *schemaSet) build(t types.Type, depth int) *schema {
	if t == nil || depth > 12 {
		return nil
	}

	switch typed := t.(type) {
	case *types.Pointer:
		return s.build(typed.Elem(), depth)
	case *types.Named:
		return s.named(typed, depth)
	case *types.Alias:
		return s.build(types.Unalias(typed), depth)
	case *types.Basic:
		return basicSchema(typed)
	case *types.Slice:
		if isByteSlice(typed) {
			return &schema{Type: "string", Format: "byte"}
		}
		return &schema{Type: "array", Items: s.build(typed.Elem(), depth+1)}
	case *types.Array:
		return &schema{Type: "array", Items: s.build(typed.Elem(), depth+1)}
	case *types.Map:
		return &schema{Type: "object", AdditionalProperties: s.build(typed.Elem(), depth+1)}
	case *types.Struct:
		return s.structSchema(typed, depth)
	case *types.Interface:
		return &schema{}
	}
	return nil
}

func (s *schemaSet) named(t *types.Named, depth int) *schema {
	obj := t.Obj()
	if obj.Pkg() == nil {
		return s.build(t.Underlying(), depth)
	}

	if obj.Pkg().Path() == "time" && obj.Name() == "Time" {
		return &schema{Type: "string", Format: "date-time"}
	}
	if obj.Pkg().Path() == "time" && obj.Name() == "Duration" {
		return &schema{Type: "string", Description: "Duration, such as 30s or 1h0m0s"}
	}
	if obj.Pkg().Path() == "github.com/google/uuid" && obj.Name() == "UUID" {
		return &schema{Type: "string", Format: "uuid"}
	}
	if obj.Pkg().Path() == "mime/multipart" && obj.Name() == "FileHeader" {
		return &schema{Type: "string", Format: "binary"}
	}

	if _, ok := t.Underlying().(*types.Struct); !ok {
		return s.build(t.Underlying(), depth)
	}

	name := instantiatedName(t)
	if !s.seen[name] {
		s.seen[name] = true
		s.byName[name] = &schema{Type: "object"}
		built := s.structSchema(t.Underlying().(*types.Struct), depth+1)
		if built != nil {
			s.byName[name] = built
		}
	}
	return &schema{Ref: "#/components/schemas/" + name}
}

func (s *schemaSet) structSchema(t *types.Struct, depth int) *schema {
	out := &schema{Type: "object", Properties: map[string]*schema{}}
	s.fields(t, depth, out)
	if len(out.Properties) == 0 {
		return &schema{Type: "object"}
	}
	return out
}

func (s *schemaSet) fields(t *types.Struct, depth int, out *schema) {
	for i := 0; i < t.NumFields(); i++ {
		field := t.Field(i)
		tag := parseTag(t.Tag(i))

		if field.Embedded() && tag.name == "" {
			if embedded, ok := underlyingStruct(field.Type()); ok {
				s.fields(embedded, depth, out)
				continue
			}
		}
		if !field.Exported() || tag.skip {
			continue
		}

		name := tag.name
		if name == "" {
			name = field.Name()
		}
		built := s.build(field.Type(), depth+1)
		if built == nil {
			continue
		}
		out.Properties[name] = built
		built.Description = tag.description
		built.Example = tag.example
		if tag.required {
			out.Required = append(out.Required, name)
		}
		if len(tag.enum) > 0 {
			built.Enum = append([]string(nil), tag.enum...)
		}
	}
}

func underlyingStruct(t types.Type) (*types.Struct, bool) {
	switch typed := t.(type) {
	case *types.Pointer:
		return underlyingStruct(typed.Elem())
	case *types.Named:
		st, ok := typed.Underlying().(*types.Struct)
		return st, ok
	case *types.Struct:
		return typed, true
	}
	return nil, false
}

type fieldTag struct {
	name        string
	skip        bool
	required    bool
	enum        []string
	description string
	example     string
}

func parseTag(raw string) fieldTag {
	tag := fieldTag{}
	parsed := reflect.StructTag(raw)

	jsonTag := parsed.Get("json")
	if jsonTag == "-" {
		tag.skip = true
		return tag
	}
	if jsonTag != "" {
		tag.name = strings.Split(jsonTag, ",")[0]
	}
	tag.description = parsed.Get("doc")
	tag.example = parsed.Get("example")
	binding := parsed.Get("binding")
	if strings.Contains(binding, "required") {
		tag.required = true
	}
	for _, rule := range strings.Split(binding, ",") {
		if values := strings.TrimPrefix(rule, "oneof="); values != rule {
			tag.enum = strings.Fields(values)
		}
	}
	return tag
}

func basicSchema(t *types.Basic) *schema {
	switch {
	case t.Info()&types.IsBoolean != 0:
		return &schema{Type: "boolean"}
	case t.Info()&types.IsInteger != 0:
		return &schema{Type: "integer"}
	case t.Info()&types.IsFloat != 0:
		return &schema{Type: "number"}
	case t.Info()&types.IsString != 0:
		return &schema{Type: "string"}
	}
	return &schema{}
}

func isByteSlice(t *types.Slice) bool {
	basic, ok := t.Elem().(*types.Basic)
	return ok && basic.Kind() == types.Byte
}

func instantiatedName(t *types.Named) string {
	obj := t.Obj()
	name := schemaName(obj.Pkg().Path(), obj.Name())
	args := t.TypeArgs()
	if args == nil || args.Len() == 0 {
		return name
	}
	parts := make([]string, 0, args.Len())
	for i := 0; i < args.Len(); i++ {
		parts = append(parts, argName(args.At(i)))
	}
	return name + "Of" + strings.Join(parts, "And")
}

func argName(t types.Type) string {
	switch typed := t.(type) {
	case *types.Pointer:
		return argName(typed.Elem())
	case *types.Slice:
		return argName(typed.Elem()) + "s"
	case *types.Named:
		return typed.Obj().Name()
	case *types.Basic:
		return strings.ToUpper(typed.Name()[:1]) + typed.Name()[1:]
	}
	return "Value"
}

func schemaName(pkgPath, name string) string {
	parts := strings.Split(pkgPath, "/")
	return parts[len(parts)-1] + "." + name
}
