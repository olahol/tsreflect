package tsreflect_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/olahol/tsreflect"
)

type MyStruct struct {
	Number int
	String int    `json:",string"`
	Alias  string `json:"alias"`
	Hidden string `json:"-"`
}

func Example_simple() {
	g := tsreflect.New()

	var x MyStruct
	typ := reflect.TypeOf(x)

	g.Add(typ)

	fmt.Println(g.DeclarationsTypeScript())
	fmt.Printf("typeof x == %s", g.TypeOf(typ))
	// Output:
	// interface MyStruct { "Number": number; "String": string; "alias": string; }
	// typeof x == MyStruct
}

type MyCustomStruct struct {
	A string
	B string
	C string
}

func (s MyCustomStruct) MarshalJSON() ([]byte, error) {
	return []byte(strings.Join([]string{s.A, s.B, s.C}, ",")), nil
}

// TypeScriptType(g *Generator, optional bool) string
func (s MyCustomStruct) TypeScriptType(*tsreflect.Generator, bool) string {
	return "string"
}

func Example_customTypeScriptType() {
	g := tsreflect.New()

	var x MyCustomStruct
	typ := reflect.TypeOf(x)

	g.Add(typ)

	fmt.Println(g.DeclarationsTypeScript())
	fmt.Printf("typeof x == %s", g.TypeOf(typ))
	// Output:
	// typeof x == string
}

func ExampleWithFlatten() {
	g := tsreflect.New(tsreflect.WithFlatten())

	var x MyStruct
	typ := reflect.TypeOf(x)

	g.Add(typ)

	fmt.Println(g.DeclarationsTypeScript())
	fmt.Printf("typeof x == %s", g.TypeOf(typ))
	// Output:
	// typeof x == { "Number": number; "String": string; "alias": string; }
}

func ExampleWithNamer() {
	g := tsreflect.New(tsreflect.WithNamer(tsreflect.PackageNamer))

	var x json.SyntaxError
	typ := reflect.TypeOf(x)

	g.Add(typ)

	fmt.Println(g.DeclarationsTypeScript())
	fmt.Printf("typeof x == %s", g.TypeOf(typ))
	// Output:
	// interface EncodingJsonSyntaxError { "Offset": number; }
	// typeof x == EncodingJsonSyntaxError
}
