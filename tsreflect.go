// Package tsreflect implements a reflection based TypeScript type generator
// for types that can be marshaled with encoding/json.
package tsreflect

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

var (
	typeOfMarshaler       = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	typeOfTypeScriptTyper = reflect.TypeOf((*TypeScriptTyper)(nil)).Elem()
	typeOfByteSlice       = reflect.TypeOf([]byte{})
	typeOfTime            = reflect.TypeOf(time.Time{})
	typeOfBigInt          = reflect.TypeOf(big.NewInt(0))
)

// TypeScriptTyper is the interface implemented by types that can serialize
// themselves into valid TypeScript types. The `optional` flag is used for
// when a type is part of an optional field in an object.
type TypeScriptTyper interface {
	TypeScriptType(g *Generator, optional bool) string
}

// Typer is a function that can serialize types in to valid TypeScript types.
// The `optional` flag is used for when a type is part of an optional field in
// an object.
//
// Used for giving TypeScript types to external types that implement custom json
// marshaling.
type Typer func(g *Generator, typ reflect.Type, optional bool) string

// Namer is a function that gives names to TypeScript types in a generator.
type Namer func(typ reflect.Type, isNameTaken func(name string) bool) string

// DefaultNamer a namer function that names types sequentially (MyStruct,
// MyStruct2, MyStruct3 ...)
func DefaultNamer(typ reflect.Type, isNameTaken func(string) bool) string {
	return sequentialNamer(typ.Name(), isNameTaken)
}

// PackageNamer a namer function which names types with their full package path
// (my_package.MyStruct => MyPackageMyStruct, other_package.MyStruct =>
// OtherPackageMyStruct ...).
func PackageNamer(typ reflect.Type, isNameTaken func(string) bool) string {
	return sequentialNamer(pkgPathName(typ.PkgPath(), typ.Name()), isNameTaken)
}

func sequentialNamer(name string, isNameTaken func(string) bool) string {
	if !isNameTaken(name) {
		return name
	}

	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%d", name, i)
		if !isNameTaken(candidate) {
			return candidate
		}
	}
}

// A Declaration is a named TypeScript type.
type Declaration struct {
	Name string
	Type string
}

// A Generator is a creator of TypeScript types and declarations.
type Generator struct {
	flatten  bool
	warnings bool
	warn     func(string, ...any)
	namer    Namer

	typers   map[reflect.Type]Typer
	types    map[reflect.Type]struct{}
	circular map[reflect.Type]struct{}
	symbols  map[reflect.Type]string
	names    map[string]reflect.Type
}

// Option is a generator option.
type Option func(*Generator)

// WithNamer sets the namer of the generator.
func WithNamer(namer Namer) Option {
	return func(g *Generator) {
		g.namer = namer
	}
}

// WithFlatten flatten output types, minimizing the amount of top-level
// declarations needed.
func WithFlatten() Option {
	return func(g *Generator) {
		g.flatten = true
	}
}

// WithNoWarnings suppress warnings.
func WithNoWarnings() Option {
	return func(g *Generator) {
		g.warnings = false
	}
}

// WithTyper add a Typer function for `typ`.
func WithTyper(typ reflect.Type, typer Typer) Option {
	return func(g *Generator) {
		g.typers[typ] = typer
	}
}

// New create a new generator with options.
func New(options ...Option) *Generator {
	g := &Generator{
		warnings: true,
		warn:     log.Printf,
		typers: map[reflect.Type]Typer{
			typeOfByteSlice: func(g *Generator, t reflect.Type, optional bool) string {
				if optional {
					return "string"
				}

				return "(string | null)"
			},
			typeOfTime: func(g *Generator, t reflect.Type, optional bool) string {
				return "string"
			},
			typeOfBigInt: func(g *Generator, t reflect.Type, optional bool) string {
				if optional {
					return "number"
				}

				return "(number | null)"
			},
		},
		types:    make(map[reflect.Type]struct{}),
		circular: make(map[reflect.Type]struct{}),
		symbols:  make(map[reflect.Type]string),
		names:    make(map[string]reflect.Type),
	}

	g.namer = DefaultNamer

	for _, option := range options {
		option(g)
	}

	return g
}

// Add add a type to the generator.
func (g *Generator) Add(typ reflect.Type) {
	g.add(typ, nil)
}

// TypeOf the TypeScript type for `typ`.
func (g *Generator) TypeOf(typ reflect.Type) string {
	return g.typeOf(typ, false)
}

// Declarations returns the required top-level declarations for TypeScript types
// in this generator.
func (g *Generator) Declarations() (ds []Declaration) {
	names := make([]string, 0, len(g.symbols))
	for _, name := range g.symbols {
		names = append(names, name)
	}

	sort.Strings(names)

	var sb strings.Builder
	for _, name := range names {
		typ := g.names[name]

		if _, ok := g.circular[typ]; !ok && g.flatten {
			continue
		}

		if g.hasCustomType(typ) {
			continue
		}

		g.writeStructDecl(&sb, typ)

		ds = append(ds, Declaration{
			Name: name,
			Type: sb.String(),
		})

		sb.Reset()
	}

	return
}

// DeclarationsTypeScript returns the needed top-level TypeScript declarations for the
// types in the generator.
func (g *Generator) DeclarationsTypeScript() string {
	return g.declarations(false)
}

// DeclarationsJSDoc returns the needed top-level JSDoc declarations for the
// types in the generator.
func (g *Generator) DeclarationsJSDoc() string {
	return g.declarations(true)
}

func (g *Generator) add(typ reflect.Type, parent reflect.Type) bool {
	if typ == nil {
		return false
	}

	if _, ok := g.types[typ]; ok {
		return typ == parent
	}

	g.types[typ] = struct{}{}

	switch typ.Kind() {
	case reflect.Array:
		return g.add(typ.Elem(), parent)
	case reflect.Slice:
		return g.add(typ.Elem(), parent)
	case reflect.Map:
		return g.add(typ.Key(), parent) || g.add(typ.Elem(), parent)
	case reflect.Pointer:
		return g.add(typ.Elem(), parent)
	case reflect.Struct:
		hasName := typ.Name() != ""
		hasExportedFields := countExportedFields(typ) > 0

		isCircular := false
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)

			if !f.IsExported() || hasTagOmit(f) {
				continue
			}

			if hasName {
				isCircular = isCircular || g.add(f.Type, typ)
			} else {
				isCircular = isCircular || g.add(f.Type, parent)
			}
		}

		if isCircular {
			g.circular[typ] = struct{}{}
		}

		if hasName && hasExportedFields {
			name := g.namer(typ, g.isNameTaken)

			if g.isNameTaken(name) {
				panic(fmt.Sprintf("tsreflect: namer returned taken name %q", name))
			}

			g.symbols[typ] = name
			g.names[name] = typ
		}

		return false
	default:
		return false
	}
}

func (g *Generator) typeOf(typ reflect.Type, optional bool) string {
	if typ == nil {
		return "any"
	}

	if typ.Implements(typeOfTypeScriptTyper) {
		t := reflect.New(typ).Elem().Interface().(TypeScriptTyper)
		return t.TypeScriptType(g, optional)
	}

	if typer, ok := g.typers[typ]; ok {
		return typer(g, typ, optional)
	}

	if typ.Implements(typeOfMarshaler) && g.warnings {
		g.warn("tsreflect: WARNING json.Marshaler implemented for type %q but no corresponding typer could be found.", typ.Name())
	}

	switch typ.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	case reflect.Array:
		elem := g.typeOf(typ.Elem(), false)

		s := make([]string, typ.Len())
		for i := range s {
			s[i] = elem
		}

		return fmt.Sprintf("[%s]", strings.Join(s, ", "))
	case reflect.Slice:
		if optional {
			return fmt.Sprintf("%s[]", g.typeOf(typ.Elem(), false))
		}

		return fmt.Sprintf("(%s[] | null)", g.typeOf(typ.Elem(), false))
	case reflect.Map:
		if optional {
			return fmt.Sprintf("{ [key in (%s)]: (%s) }", g.typeOf(typ.Key(), false), g.typeOf(typ.Elem(), false))
		}

		return fmt.Sprintf("({ [key in (%s)]: (%s) } | null)", g.typeOf(typ.Key(), false), g.typeOf(typ.Elem(), false))
	case reflect.Pointer:
		if optional {
			return g.typeOf(typ.Elem(), false)
		}

		return fmt.Sprintf("(%s | null)", g.typeOf(typ.Elem(), false))
	case reflect.Struct:
		name := g.symbols[typ]
		_, isCircular := g.circular[typ]

		if name == "" || (!isCircular && g.flatten) {
			var sb strings.Builder
			g.writeStructDecl(&sb, typ)
			return sb.String()
		}

		return name
	case reflect.Interface:
		return "any"
	default:
		return ""
	}
}

func (g *Generator) declarations(jsDoc bool) string {
	var sb strings.Builder

	decls := g.Declarations()
	for i, decl := range decls {
		if jsDoc {
			sb.WriteString("/** @typedef {")
		} else {
			sb.WriteString(fmt.Sprintf("interface %s ", decl.Name))
		}

		sb.WriteString(decl.Type)

		if jsDoc {
			sb.WriteString(fmt.Sprintf("} %s */", decl.Name))
		}

		if i < len(decls)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (g *Generator) writeStructDecl(sb *strings.Builder, typ reflect.Type) {
	sb.WriteString("{ ")

	g.writeStructFields(sb, typ)

	sb.WriteString("}")
}

func (g *Generator) writeStructFields(sb *strings.Builder, typ reflect.Type) {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)

		if !f.IsExported() || hasTagOmit(f) {
			continue
		}

		if f.Anonymous {
			g.writeStructFields(sb, f.Type)
		} else {
			sb.WriteString(g.structField(f))
			sb.WriteString("; ")
		}
	}
}

func hasTagOmit(f reflect.StructField) bool {
	if tag, ok := f.Tag.Lookup("json"); ok && tag == "-" {
		return true
	}

	return false
}

func (g *Generator) structField(f reflect.StructField) string {
	name := f.Name
	omit := false

	var typ string
	if tag, ok := f.Tag.Lookup("json"); ok {
		if !strings.ContainsRune(tag, ',') {
			name = tag
		} else {
			parts := strings.Split(tag, ",")

			if parts[0] != "" {
				name = parts[0]
			}
			switch parts[1] {
			case "string":
				typ = "string"
			case "omitempty":
				omit = true
			}
		}
	}

	if typ == "" {
		typ = g.typeOf(f.Type, omit)
	}

	if omit {
		return fmt.Sprintf("%q?: %s", name, typ)
	}

	return fmt.Sprintf("%q: %s", name, typ)
}

func countExportedFields(typ reflect.Type) int {
	if typ.Kind() != reflect.Struct {
		return 0
	}

	var count int
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)

		if !f.IsExported() || hasTagOmit(f) {
			continue
		}

		if f.Anonymous {
			count += countExportedFields(f.Type)
		} else {
			count += 1
		}
	}

	return count
}

func (g *Generator) hasCustomType(typ reflect.Type) bool {
	_, ok := g.typers[typ]

	return ok || typ.Implements(typeOfTypeScriptTyper)
}

func (g *Generator) isNameTaken(name string) bool {
	_, ok := g.names[name]

	return ok
}

func title(s string) string {
	if s == "" {
		return ""
	}

	rs := []rune(s)
	rs[0] = unicode.ToUpper(rs[0])
	return string(rs)
}

func pascalCase(s string) string {
	re := regexp.MustCompile(`([._-]|\s)+`)

	parts := re.Split(s, -1)
	for i, part := range parts {
		parts[i] = title(part)
	}

	return strings.Join(parts, "")
}

func pkgPathName(pkgPath string, name string) string {
	if pkgPath == "" {
		return name
	}

	var parts []string
	for _, segment := range strings.Split(pkgPath, "/") {
		if strings.ContainsRune(segment, '.') {
			continue
		}

		part := pascalCase(segment)

		if part == "" {
			continue
		}

		parts = append(parts, part)
	}

	return fmt.Sprintf("%s%s", strings.Join(parts, ""), name)
}
