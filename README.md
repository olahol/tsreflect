# tsreflect

[![Go Report Card](https://goreportcard.com/badge/github.com/olahol/tsreflect)](https://goreportcard.com/report/github.com/olahol/tsreflect)
[![GoDoc](https://godoc.org/github.com/olahol/tsreflect?status.svg)](https://godoc.org/github.com/olahol/tsreflect)

> Flexible reflection based TypeScript type generator for Go types that can be marshalled with `encoding/json`.

## Install

```bash
go get github.com/olahol/tsreflect
```

## Example

```go
package main

import (
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

func main() {
	g := tsreflect.New()

	var x MyStruct
	typ := reflect.TypeOf(x)

	g.Add(typ)

	fmt.Println(g.Declarations())
	fmt.Printf("typeof x == %s", g.Type(typ))
	// Output:
	// interface MyStruct { "Number": number; "String": string; "alias": string; }
	// typeof x == MyStruct
}
```