# gompare

[![PkgGoDev](https://pkg.go.dev/badge/github.com/chriss-de/gompare/v2)](https://pkg.go.dev/github.com/chriss-de/gompare/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/chriss-de/gompare)](https://goreportcard.com/report/github.com/chriss-de/gompare)
[![CI](https://github.com/chriss-de/gompare/actions/workflows/ci.yml/badge.svg)](https://github.com/chriss-de/gompare/actions/workflows/ci.yml)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](LICENSE)

Compare two Go values of the same type and get a list of what was **added**, **changed** or **removed** -
with a path to every difference. Nested structs, slices, arrays, maps, pointers and interfaces are
walked recursively. Slice elements can be matched by an identifier instead of their position.

```
go get github.com/chriss-de/gompare/v2
```

Requires Go 1.24 or newer.

## Quick start

```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/chriss-de/gompare/v2"
)

type Item struct {
	SKU string `cmp:"sku,identifier"`
	Qty int    `cmp:"qty"`
}

type Order struct {
	ID    string            `cmp:"id"`
	Items []Item            `cmp:"items"`
	Tags  []string          `cmp:"tags"`
	Meta  map[string]string `cmp:"meta"`
	Note  string            `cmp:"-"`
}

func main() {
	left := Order{
		ID:    "1234",
		Items: []Item{{SKU: "apple", Qty: 1}, {SKU: "pear", Qty: 2}},
		Tags:  []string{"new"},
		Meta:  map[string]string{"channel": "web"},
		Note:  "ignored",
	}
	right := Order{
		ID:    "1234",
		Items: []Item{{SKU: "apple", Qty: 3}, {SKU: "plum", Qty: 1}},
		Tags:  []string{"new", "paid"},
		Meta:  map[string]string{"channel": "shop"},
		Note:  "also ignored",
	}

	diffs, err := gompare.Compare(left, right)
	if err != nil {
		panic(err)
	}

	for _, d := range diffs {
		out, _ := json.Marshal(d)
		fmt.Println(string(out))
	}
}
```

Output:

```json
{"type":"changed","path":["items","apple","qty"],"left":1,"right":3}
{"type":"removed","path":["items","pear","sku"],"left":"pear","right":null}
{"type":"removed","path":["items","pear","qty"],"left":2,"right":null}
{"type":"added","path":["items","plum","sku"],"left":null,"right":"plum"}
{"type":"added","path":["items","plum","qty"],"left":null,"right":1}
{"type":"added","path":["tags","1"],"left":null,"right":"paid"}
{"type":"changed","path":["meta","channel"],"left":"web","right":"shop"}
```

The items are matched by their `sku` identifier, not by position. The `Note` field is excluded by its `-` tag.
The same program is the `ExampleCompare` test in this repository, so the output above is verified on every test run.

## How it works

The comparison is always between **LEFT** and **RIGHT**. Think of two sheets of paper in front of you:
whatever is only on the right sheet was *added*, whatever is only on the left sheet was *removed*,
and whatever is on both sheets but different was *changed*.

`Compare` returns `Differences`, a slice of:

```go
type Difference struct {
	Type  DiffType // ADDED, CHANGED or REMOVED (JSON: "added", "changed", "removed")
	Path  []string // field names, map keys, slice indexes or identifiers along the way
	Left  any      // the value on the left side, nil if added
	Right any      // the value on the right side, nil if removed
}
```

`Left` and `Right` always hold the value in the type of its field. Unexported struct fields are
compared like exported ones. `time.Time` values are compared by instant, not by location.

### Reusing a Comparer

Options can be passed to `Compare` directly or to `NewComparer` once. A `Comparer` holds no state
between calls and can be used from multiple goroutines.

```go
cmp, err := gompare.NewComparer(gompare.WithSliceOrdering(), gompare.WithEmbeddedStructsAsField())
if err != nil {
	panic(err)
}
diffs, err := cmp.Compare(left, right)
```

## Tags

Struct fields are configured with the `cmp` tag. The first value is the name used in the path
(the Go field name if empty), everything after a comma is an option.

| Tag                 | Effect                                                                    |
|---------------------|---------------------------------------------------------------------------|
| `cmp:"name"`        | use `name` in the path instead of the field name                           |
| `cmp:"-"`           | exclude the field from the comparison                                      |
| `cmp:",identifier"` | use the field to match elements of a slice or array (see below)            |

`WithTagName("json")` lets you reuse another tag. Note that linters such as staticcheck (check SA5008)
flag `identifier` as an unknown option on well known tags like `json` or `xml`. Keep the default `cmp`
tag if you want to stay lint clean.

### Identifiers

By default slice elements are matched by equality, regardless of their position. If the elements are
structs and one field is tagged as `identifier`, elements with the same identifier on both sides are
compared with each other, and the identifier is used in the path instead of the index.

Several fields can make up the identifier. They are joined with `|` by default
(`WithCombinedIdentifierJoinString` changes the separator):

```go
type Seat struct {
	Row    string `cmp:"row,identifier"`
	Number int    `cmp:"number,identifier"`
	Owner  string `cmp:"owner"`
}
// path of a changed owner: ["A|12", "owner"]
```

For full control the identifier can be a Go template. Every identifier field is available under its
Go field name and under its tag name. All identifier fields of a struct must carry the same template:

```go
type Seat struct {
	Row    string `cmp:"row,identifier:{{ .row }}-{{ .number }}"`
	Number int    `cmp:"number,identifier:{{ .row }}-{{ .number }}"`
	Owner  string `cmp:"owner"`
}
// path of a changed owner: ["A-12", "owner"]
```

A template that does not parse, references an unknown key or differs between fields results in
`ErrIdentifierTemplate`. The template may contain `:` but not `,`.

Rules:

- An identifier must be unique within one slice, otherwise `Compare` returns `ErrDuplicateIdentifier`.
- Elements without an identifier (for example a `nil` pointer in a `[]*Seat`) are matched by their index.
- The first struct element found on either side decides whether a slice is compared by identifier.

## Options

| Option                                        | Effect                                                                                                                                                                                                                 |
|-----------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `WithTagName(name)`                           | read field names and options from this struct tag instead of `cmp`                                                                                                                                                     |
| `WithSliceOrdering()`                         | compare slice elements by position. Without it an element counts as present if an equal element exists anywhere on the other side; elements left over on both sides are then paired by index and compared with each other |
| `WithCombinedIdentifierJoinString(sep)`       | separator for combined identifiers in the path, default `\|`                                                                                                                                                           |
| `WithEmbeddedStructsAsField()`                | list an embedded struct under its own name instead of merging its fields into the parent                                                                                                                               |
| `WithSummarizeMissingStructs()`               | report a struct that exists on one side only as one entry holding the whole struct                                                                                                                                     |
| `WithSummarizeMissing()`                      | like above, but for every container: struct, slice, array and map                                                                                                                                                      |
| `WithStructMapKeys()`                         | encode map keys that are not strings, numbers or bools with gob and base64 for the path. Without it they are rendered with `fmt.Sprint`                                                                                  |
| `WithAllowTypeMismatch()`                     | report two values of different kind (an `int` that became a `string` in a `map[string]any`) as `changed` instead of returning `ErrTypeMismatch`                                                                        |
| `WithAllowDifferentStructs()`                 | compare structs of different types field by field (matched by Go field name) instead of returning `ErrTypeMismatch`                                                                                                    |

## Missing values

If a value exists on one side only, containers are listed by their content: a struct by its fields
(including fields holding their zero value), a slice or array by its elements, a map by its entries.
This also applies to the target of a pointer or interface that is `nil` on one side. A scalar behind a
`nil` pointer or interface is one `changed` entry holding the dereferenced value. `nil` pointers, `nil`
interfaces and empty containers inside a missing struct produce no entry.

`WithSummarizeMissing()` and `WithSummarizeMissingStructs()` switch to one entry per missing container.

## Filtering differences

`GetDifferences` returns an iterator over the differences that pass all given filters.
`HasDifferences` reports whether any difference passes them.

```go
for d := range diffs.GetDifferences(gompare.WherePathAt("items", 0), gompare.WhereDiffType(gompare.CHANGED)) {
	fmt.Printf("%s: %v -> %v\n", d.Path, d.Left, d.Right)
}
if diffs.HasDifferences(gompare.WherePath("tags")) {
	// ...
}
```

| Filter                       | Passes if                                                        |
|------------------------------|------------------------------------------------------------------|
| `WherePath(p)`               | any path segment equals `p`                                      |
| `WherePathAt(p, idx)`        | the path segment at index `idx` equals `p`                       |
| `WherePathDepth(n)`          | the path has exactly `n` segments                                |
| `WherePathDepthGt(n)`        | the path has more than `n` segments                              |
| `WherePathDepthLt(n)`        | the path has fewer than `n` segments                             |
| `WhereDiffType(t)`           | the difference is of type `t`                                    |
| `WhereOr(filters...)`        | any of the given filters passes (filters are AND-ed otherwise)   |

## Errors

`Compare` returns an error if it cannot produce a reliable result. Errors are wrapped with the path
where they happened, so match them with `errors.Is`:

| Error                    | Reason                                                                                     |
|--------------------------|--------------------------------------------------------------------------------------------|
| `ErrTypeMismatch`        | left and right are of different kind or different struct type (see the `WithAllow*` options) |
| `ErrUnsupportedType`     | a value of kind func, chan, complex or unsafe pointer was found                             |
| `ErrDuplicateIdentifier` | an identifier appears more than once within one slice                                      |
| `ErrIdentifierTemplate`  | an identifier template does not parse, references an unknown key or differs between fields |
| `ErrInvalidOption`       | an option was given an invalid value, for example an empty tag name                        |
| `ErrUnexportedField`     | unexported fields cannot be read on this Go version (the package checks this at startup)   |

## Development

```
go test -race -cover ./...
go test -bench . -run xxx
```

The CI workflow runs gofmt, vet, the race tests and staticcheck on every push.

## History and name

The original idea comes from [r3labs/diff](https://github.com/r3labs/diff). Since that project seemed
unmaintained, gompare started as a fork and became a rewrite that kept most of the original tests.

The name is a play on *go* and *compare*. It also sounds funny in German: pronounce *compare* with a
Saxon accent.

## Versioning and license

This project follows [Semantic Versioning](https://semver.org/). Breaking changes are listed in
[CHANGELOG.md](CHANGELOG.md). Licensed under the [Mozilla Public License 2.0](LICENSE).
