# tsreflect

![Build Status](https://github.com/olahol/tsreflect/actions/workflows/test.yml/badge.svg)
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

	value, _ := json.Marshal(x)

	fmt.Println(g.DeclarationsTypeScript())
	fmt.Printf("const x: %s = %s", g.TypeOf(typ), value)
	// Output:
	// interface MyStruct { "Number": number; "String": string; "alias": string; }
	// const x: MyStruct = {"Number":0,"String":"0","alias":""}
}
```
