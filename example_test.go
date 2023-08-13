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

	value, _ := json.Marshal(x)

	fmt.Println(g.DeclarationsTypeScript())
	fmt.Printf("const x: %s = %s", g.TypeOf(typ), value)
	// Output:
	// interface MyStruct { "Number": number; "String": string; "alias": string; }
	// const x: MyStruct = {"Number":0,"String":"0","alias":""}
}

type MyCustomStruct struct {
	A string
	B string
	C string
}

func (s MyCustomStruct) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", strings.Join([]string{s.A, s.B, s.C}, ","))), nil
}

// TypeScriptType(g *Generator, optional bool) string
func (s MyCustomStruct) TypeScriptType(*tsreflect.Generator, bool) string {
	return "string"
}

func Example_customTypeScriptType() {
	g := tsreflect.New()

	x := MyCustomStruct{
		A: "1",
		B: "2",
		C: "3",
	}
	typ := reflect.TypeOf(x)

	g.Add(typ)

	value, _ := json.Marshal(x)

	fmt.Println(g.DeclarationsTypeScript())
	fmt.Printf("const x: %s = %s", g.TypeOf(typ), value)
	// Output:
	// const x: string = "1,2,3"
}

func ExampleWithFlatten() {
	g := tsreflect.New(tsreflect.WithFlatten())

	var x MyStruct
	typ := reflect.TypeOf(x)

	g.Add(typ)

	value, _ := json.Marshal(x)

	fmt.Println(g.DeclarationsTypeScript())
	fmt.Printf("const x: %s = %s", g.TypeOf(typ), value)
	// Output:
	// const x: { "Number": number; "String": string; "alias": string; } = {"Number":0,"String":"0","alias":""}
}

func ExampleWithNamer() {
	g := tsreflect.New(tsreflect.WithNamer(tsreflect.PackageNamer))

	var x json.SyntaxError
	typ := reflect.TypeOf(x)

	g.Add(typ)

	value, _ := json.Marshal(x)

	fmt.Println(g.DeclarationsTypeScript())
	fmt.Printf("const x: %s = %s", g.TypeOf(typ), value)
	// Output:
	// interface EncodingJsonSyntaxError { "Offset": number; }
	// const x: EncodingJsonSyntaxError = {"Offset":0}
}
