package tsreflect

import (
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"
	"unsafe"
)

var isDebug = os.Getenv("DEBUG") != ""

func AssertNoError(t *testing.T, err error) {
	t.Helper()

	if err != nil {
		t.Error(err)
	}
}

func AssertError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Error(errors.New("should be error"))
	}
}

func AssertEqual[T comparable](t *testing.T, a, b T) {
	t.Helper()

	if a != b {
		t.Error(fmt.Errorf("%v != %v", a, b))
	}
}

func programOfGenerator[T any](g *Generator, v T) (string, error) {
	typ := reflect.TypeOf(v)
	decls := g.DeclarationsTypeScript()
	typing := g.TypeOf(typ)

	value, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	if decls == "" {
		return fmt.Sprintf("const test: %s = %s", typing, value), nil
	}

	return fmt.Sprintf("%s\nconst test: %s = %s", decls, typing, value), nil
}

func programOfValue[T any](v T, os ...Option) (string, error) {
	g := New(os...)
	g.Add(reflect.TypeOf(v))

	return programOfGenerator(g, v)
}

func typecheckSource(source string) error {
	file := fmt.Sprintf("typecheck-%d.ts", rand.Int())

	err := os.WriteFile(file, []byte(source), 0600)

	if err != nil {
		return err
	}

	defer os.Remove(file)

	bs, err := exec.Command("tsc", "--noEmit", file).Output()

	if err != nil {
		return fmt.Errorf("%s:\n\n%s", bs, source)
	}

	return nil
}

func typecheckValue[T any](v T, os ...Option) error {
	source, err := programOfValue(v, os...)

	if err != nil {
		return err
	}

	if isDebug {
		fmt.Println(source)
	}

	return typecheckSource(source)
}

func TestBool(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		x := true

		AssertNoError(t, typecheckValue(x))
	})
}

func TestInterface(t *testing.T) {
	t.Run("reflect.Interface", func(t *testing.T) {
		type S struct {
			A interface{}
		}

		var x S

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("nil", func(t *testing.T) {
		var x interface{}

		AssertNoError(t, typecheckValue(x))
	})
}

func TestNumbers(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		x := 99

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("uint", func(t *testing.T) {
		x := uint(99.0)

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("float64", func(t *testing.T) {
		x := 99.0

		AssertNoError(t, typecheckValue(x))
	})
}

func TestStrings(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		x := "test"

		AssertNoError(t, typecheckValue(x))
	})
}

func TestPointers(t *testing.T) {
	t.Run("nil pointer", func(t *testing.T) {
		var x *int

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		i := 99
		x := &i

		AssertNoError(t, typecheckValue(x))
	})
}
func TestArrays(t *testing.T) {
	t.Run("array", func(t *testing.T) {
		var x [9]int

		AssertNoError(t, typecheckValue(x))
	})
}

func TestSlices(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		var x []int

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("non-nil slice", func(t *testing.T) {
		x := make([]int, 9)

		AssertNoError(t, typecheckValue(x))
	})
}

func TestMaps(t *testing.T) {
	t.Run("nil map", func(t *testing.T) {
		var x map[int]int

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("non-nil map", func(t *testing.T) {
		x := map[int]int{
			1: 2,
		}

		AssertNoError(t, typecheckValue(x))
	})
}

func TestStructs(t *testing.T) {
	t.Run("anonymous struct", func(t *testing.T) {
		var x struct {
			A string
			B int
			C float64
		}

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("anonymous nested struct", func(t *testing.T) {
		var x struct {
			A string
			B int
			C struct {
				D int
				E float32
			}
		}

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("array of anonymous structs", func(t *testing.T) {
		var x [9]struct {
			A string
			B int
		}

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("slice of anonymous structs", func(t *testing.T) {
		x := make([]struct {
			A string
			B int
		}, 10)

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("embedded structs", func(t *testing.T) {
		type S1 struct {
			I int
		}

		type S2 struct {
			S1
		}

		var x S2

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("cyclical struct", func(t *testing.T) {
		type c struct {
			A int
			R *c
		}

		x := c{
			A: 1,
			R: &c{
				A: 2,
				R: nil,
			},
		}

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("cyclical slice struct", func(t *testing.T) {
		type c struct {
			A int
			R []c
		}

		x := c{
			A: 1,
			R: []c{{
				A: 2,
				R: nil,
			}},
		}

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("nested cyclical struct", func(t *testing.T) {
		type S1 struct {
			A int
			R *S1
		}

		type S2 struct {
			A int
			R S1
		}

		x := S2{
			A: 1,
			R: S1{
				A: 2,
				R: &S1{
					A: 9,
					R: nil,
				},
			},
		}

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("two cyclical structs", func(t *testing.T) {
		type S1 struct {
			A int
			R *S1
		}

		type S2 struct {
			A int
			R *S2
		}

		type S3 struct {
			A S1
			B S2
		}

		x := S3{
			A: S1{
				A: 1,
				R: nil,
			},
			B: S2{
				A: 2,
				R: nil,
			},
		}

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("name struct tags", func(t *testing.T) {
		type S struct {
			A int    `json:"a"`
			B string `json:"b"`
			C string `json:"-,"`
		}

		var x S

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("string struct tags", func(t *testing.T) {
		type S struct {
			A int   `json:"a"`
			B int64 `json:"b,string"`
		}

		var x S

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("omitempty struct tags", func(t *testing.T) {
		type S1 struct {
			A int  `json:"a"`
			B *int `json:"b,omitempty"`
		}

		type S2 struct {
			A int   `json:"a"`
			B []int `json:"b,omitempty"`
		}

		type S3 struct {
			A int         `json:"a"`
			B map[int]int `json:"b,omitempty"`
		}

		type S4 struct {
			A int `json:"a"`
			B *S3 `json:"b,omitempty"`
		}

		type S5 struct {
			A int `json:"a"`
			B int `json:"b,omitempty"`
		}

		var x S1
		var y S2
		var z S3
		var q S4
		var f S5

		AssertNoError(t, typecheckValue(x))
		AssertNoError(t, typecheckValue(y))
		AssertNoError(t, typecheckValue(z))
		AssertNoError(t, typecheckValue(q))
		AssertNoError(t, typecheckValue(f))
	})

	t.Run("struct name collision", func(t *testing.T) {
		g := New()

		type S struct {
			A int
			B *int
		}

		var x S

		g.Add(reflect.TypeOf(x))

		{
			type S struct {
				C int
				D *int
			}

			var x S

			g.Add(reflect.TypeOf(x))
		}

		source, err := programOfGenerator(g, x)

		AssertNoError(t, err)
		AssertNoError(t, typecheckSource(source))
	})

	t.Run("circular struct name collision", func(t *testing.T) {
		g := New()

		type S struct {
			A int
			B *S
		}

		var x S

		g.Add(reflect.TypeOf(x))

		{
			type S struct {
				C int
				D *S
			}

			var x S

			g.Add(reflect.TypeOf(x))
		}

		source, err := programOfGenerator(g, x)

		AssertNoError(t, err)
		AssertNoError(t, typecheckSource(source))
	})
}

func TestUnsupported(t *testing.T) {
	t.Run("complex64", func(t *testing.T) {
		x := complex64(10 + 20i)

		AssertError(t, typecheckValue(x))
	})

	t.Run("complex128", func(t *testing.T) {
		x := 10 + 20i

		AssertError(t, typecheckValue(x))
	})

	t.Run("chan", func(t *testing.T) {
		x := make(chan struct{})

		AssertError(t, typecheckValue(x))
	})

	t.Run("func", func(t *testing.T) {
		x := func() {}

		AssertError(t, typecheckValue(x))
	})

	t.Run("unsafe pointer", func(t *testing.T) {
		i := 10
		x := unsafe.Pointer(&i)

		AssertError(t, typecheckValue(x))
	})
}

type StringUnion string

func (s StringUnion) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%q", s)), nil
}

func (StringUnion) TypeScriptType(*Generator, bool) string {
	return fmt.Sprintf("%q | %q", "test1", "test2")
}

type NonNullSlice[T any] []T

func (s NonNullSlice[T]) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("[]"), nil
	}

	return json.Marshal([]T(s))
}

func (s NonNullSlice[T]) TypeScriptType(g *Generator, optional bool) string {
	typ := reflect.TypeOf(s)

	return fmt.Sprintf("Array<%s>", typ.Elem())
}

type Base32Slice []byte

func (s Base32Slice) MarshalJSON() ([]byte, error) {
	out := base32.StdEncoding.EncodeToString(s)

	return []byte(fmt.Sprintf("%q", out)), nil
}

func (s Base32Slice) TypeScriptType(*Generator, bool) string {
	return "string"
}

func TestCustomTypes(t *testing.T) {
	t.Run("union", func(t *testing.T) {
		var x StringUnion

		AssertError(t, typecheckValue(x))

		x = "test2"

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("generics", func(t *testing.T) {
		var x NonNullSlice[string]

		AssertNoError(t, typecheckValue(x))

		x = append(x, "test")

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("transform", func(t *testing.T) {
		var x Base32Slice

		AssertNoError(t, typecheckValue(x))

		x = []byte("test")

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("with typer", func(t *testing.T) {
		type S struct {
			A string
		}

		var x S

		typ := reflect.TypeOf(x)

		AssertNoError(t, typecheckValue(x, WithTyper(typ, func(g *Generator, typ reflect.Type, optional bool) string {
			return "Record<string, string>"
		})))

		AssertNoError(t, typecheckValue(x))
	})
}

func TestBuiltin(t *testing.T) {
	t.Run("rune", func(t *testing.T) {
		//x := '⌘'
		//x := []rune{'a', 'b', 'c'}
		var x []rune

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("[]byte should be typed as string | null", func(t *testing.T) {
		var x []byte

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("time.Time should be typed as string", func(t *testing.T) {
		var x time.Time

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("big.Int should be typed as 'number | null'", func(t *testing.T) {
		x := big.NewInt(99)

		AssertNoError(t, typecheckValue(x))
	})
}

type Marshaled struct {
	A int
}

func (Marshaled) MarshalJSON() ([]byte, error) {
	return []byte("string"), nil
}

func TestWarning(t *testing.T) {
	t.Run("should warn of missing typer", func(t *testing.T) {
		var x Marshaled

		g := New()
		typ := reflect.TypeOf(x)

		var called bool
		g.warn = func(s string, a ...any) {
			called = true
		}

		g.Add(typ)
		g.TypeOf(typ)

		AssertEqual(t, called, true)
	})

	t.Run("should not warn of missing typer", func(t *testing.T) {
		var x Marshaled

		g := New(WithNoWarnings())
		typ := reflect.TypeOf(x)

		var called bool
		g.warn = func(s string, a ...any) {
			called = true
		}

		g.Add(typ)
		g.TypeOf(typ)

		AssertEqual(t, called, false)
	})
}

func TestNamer(t *testing.T) {
	t.Run("camel case", func(t *testing.T) {
		AssertEqual(t, pascalCase("domain.name"), "DomainName")
		AssertEqual(t, pascalCase("snake_case"), "SnakeCase")
		AssertEqual(t, pascalCase("kebab-case"), "KebabCase")
		AssertEqual(t, pascalCase("camelCase"), "CamelCase")
		AssertEqual(t, pascalCase("PascalCase"), "PascalCase")
		AssertEqual(t, pascalCase("Space Name"), "SpaceName")
		AssertEqual(t, pascalCase("path/Case/Name"), "Path/Case/Name")
		AssertEqual(t, pascalCase("mixed_case-name"), "MixedCaseName")
		AssertEqual(t, pascalCase("mixed case-name"), "MixedCaseName")
		AssertEqual(t, pascalCase("mixed.case___name.kebab-----case..com"), "MixedCaseNameKebabCaseCom")
		AssertEqual(t, pascalCase(".."), "")
		AssertEqual(t, pascalCase("..relativeName"), "RelativeName")
	})

	t.Run("package path name", func(t *testing.T) {
		AssertEqual(t, pkgPathName("github.com/olahol/tsreflect", "Generator"), "OlaholTsreflectGenerator")
		AssertEqual(t, pkgPathName("github.com/shopspring/decimal", "Decimal"), "ShopspringDecimalDecimal")
		AssertEqual(t, pkgPathName("encoding/json", "Decoder"), "EncodingJsonDecoder")
		AssertEqual(t, pkgPathName("../test", "Struct"), "TestStruct")
		AssertEqual(t, pkgPathName("snake_case", "Struct_Name"), "SnakeCaseStruct_Name")
		AssertEqual(t, pkgPathName("empty//part", "Name"), "EmptyPartName")
		AssertEqual(t, pkgPathName("", "Name"), "Name")
	})
}

func TestCoverage(t *testing.T) {
	t.Run("optional byte slice", func(t *testing.T) {
		type S struct {
			A []byte `json:",omitempty"`
		}

		var x S

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("optional bigint", func(t *testing.T) {
		type S struct {
			A *big.Int `json:",omitempty"`
		}

		var x S

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("jsdoc declarations", func(t *testing.T) {
		type S struct {
			A string `json:"a"`
		}

		var x S

		g := New()
		g.Add(reflect.TypeOf(x))

		AssertEqual(t, g.DeclarationsJSDoc(), `/** @typedef {{ "a": string; }} S */`)
	})

	t.Run("bad namer", func(t *testing.T) {
		defer func() {
			recover()
		}()

		type S1 struct {
			A string `json:"a"`
		}

		type S2 struct {
			A string `json:"a"`
		}

		var x S1
		var y S2

		g := New(WithNamer(func(typ reflect.Type, isNameTaken func(name string) bool) string {
			return "Name"
		}))
		g.Add(reflect.TypeOf(x))
		g.Add(reflect.TypeOf(y))
	})

	t.Run("bad namer", func(t *testing.T) {
		defer func() {
			recover()
		}()

		type S1 struct {
			A string `json:"a"`
		}

		type S2 struct {
			A string `json:"a"`
		}

		var x S1
		var y S2

		g := New(WithNamer(func(typ reflect.Type, isNameTaken func(name string) bool) string {
			return "Name"
		}))
		g.Add(reflect.TypeOf(x))
		g.Add(reflect.TypeOf(y))
	})
}

type Date time.Time

func (d Date) MarshalJSON() ([]byte, error) {
	return time.Time(d).MarshalJSON()
}

func (Date) TypeScriptType(g *Generator, optional bool) string {
	return "Date"
}

func TestBugs(t *testing.T) {
	t.Run("omitted fields should not create semicolons", func(t *testing.T) {
		type S struct {
			A bool `json:"-"`
			B bool
		}

		var x S

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("non struct anonymous types should work", func(t *testing.T) {
		type S struct {
			A int
			B int
			StringUnion
		}

		x := S{
			StringUnion: "test1",
		}

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("byte slices should have type (string | null)", func(t *testing.T) {
		var x []byte

		AssertNoError(t, typecheckValue(x))

		x = []byte("test")

		AssertNoError(t, typecheckValue(x))
	})

	t.Run("should not declare empty types", func(t *testing.T) {
		var x time.Time

		g := New()
		g.Add(reflect.TypeOf(x))

		AssertEqual(t, g.DeclarationsTypeScript(), "")
	})

	t.Run("should not declare empty types", func(t *testing.T) {
		var x time.Time

		g := New()
		g.Add(reflect.TypeOf(x))

		AssertEqual(t, g.DeclarationsTypeScript(), "")
	})

	t.Run("should handle automatic dereferencing and interfaces", func(t *testing.T) {
		var x *Date

		typ := reflect.TypeOf(x)

		g := New()
		g.Add(typ)

		AssertEqual(t, "(Date | null)", g.TypeOf(typ))
	})
}
