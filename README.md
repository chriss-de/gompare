# gompare [![PkgGoDev](https://pkg.go.dev/badge/github.com/chriss-de/gompare)](https://pkg.go.dev/github.com/chriss-de/gompare) [![Go Report Card](https://goreportcard.com/badge/github.com/chriss-de/gompare)](https://goreportcard.com/report/github.com/chriss-de/gompare)

A go module for comparing structures and other go types of the same type.

It uses reflection and produces a list of Differences that show the difference between the 2 objects.

## History, heritage and name
The original idea and project comes from project [diff](https://github.com/r3labs/diff). Since the project seems dead and/or unmaintained 
I kind of forked and kind of rewrote the whole thing. I reused almost all tests since the basic idea is the same.

I needed a library that can compare two structs with nested fields (including slices) in my other projects and therefore worked on gompare.

About the name - it's a play on words - go and compare.
It also sounds funny in german as you can pronounce it like compare but with a saxon dialect.


## Basic idea

The comparison is always between LEFT and RIGHT. Think of it like two papers in front of you and you compare those two.

## Difference Format

When comparing two structures using `Compare`, it produces `Differences` - a list of `Difference` structs.
Any detected difference will be noted:

```go
type Difference struct {
	Type  DiffType    // The type of change detected; one of: added , changed , removed
	Path  []string    // The path of the detected change; will contain any field name or array index that was part of the traversal
	Left  any         // The value on the left side
	Right any         // The value on the right side 
}
```

Given the example below, we are comparing two slices where the third element (index=2) has been removed:

```go
left := []int{1, 2, 3, 4}
right := []int{1, 2, 4}

differences, _ := gompare.Compare(left, right)
```

The result should be:

```go
Difference{
    Type:   gompare.REMOVED,
    Path:   ["2"],
    Left:   3,
    Right:  nil,
}
```

### Tags

All tag values are prefixed with `cmp`. i.e. `cmp:"name"`.

| Tag           | Usage                                                                                                                                                                                                                                                                |
|---------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `-`           | Excludes a value from being compared                                                                                                                                                                                                                                 |
| `,identifier` | If you need to compare arrays/slices by a matching identifier and not based on order, you can specify the `identifier` tag. If an identifiable element is found in both the left and right structure, they will be directly compared. i.e. `cmp:"name,identifier"`   |

### Identifier

gompare supports combined identifier. If your struct has two fields that make up a unique ID you can tag it like that.

```go
type MyStruct struct {
	Name    string  `cmp:"name,identifier"`
	Attr1   int     `cmp:"attr1,identifier"`
	Attr2   int     `cmp:"attr2"`
}
```

When a difference is found the path will use both keys combined - separated by `|` (default)

IF needed you can go fancy with go templating

```go
type MyStruct struct {
	Name    string  `cmp:"name,identifier={{ .name }}-i-am-the-key-{{ .attr1 }}"`
	Attr1   int     `cmp:"attr1,identifier={{ .name }}-i-am-the-key-{{ .attr1 }}"`
	Attr2   int     `cmp:"attr2"`
}
```

For this to work both identifier must have the same template string. 

## Usage

Comparing a basic struct can be accomplished using the Compare functions.

```go
import "github.com/chriss-de/gompare"

type MyStruct struct {
    ID    string `cmp:"id"`
    Items []int  `cmp:"items"`
}

func main() {
    left := Order{ ID: "1234", Items: []int{1, 2, 3, 4} }

    right := Order{ ID: "1234", Items: []int{1, 2, 4} }

    differences, err := gompare.Compare(left, right)
}
```

In this example, the output generated will indicate that the third 
element with a value of '3' was removed from items. When marshalling 
to json, the output will look like:

```json
[
    {
        "type": "removed",
        "path": ["items", "2"],
        "left": 3,
        "right": null
    }
]
```

## Configuration

Options can be set on the differ at call time which effect how diff acts when 
building the change log.

```go
import "github.com/chriss-de/gompare"

func main() {
    ...
    diffs, err := gompare.Compare(
		left, 
		right,
		gompare.WithSliceOrdering(),
		gompare.WithEmbeddedStructsAsField(), 
	)
    ...
    cmp, err := gompare.NewComparer(
		gompare.WithSliceOrdering(),
		gompare.WithEmbeddedStructsAsField(),
	)
    if err != nil {
        panic(err)
    }
    diffs, err := cmp.Compare(left, right)
	...
}
```

Available options are:

`WithTagName(name string)` uses this name as tag to look for on struct fields

`WithSliceOrdering()` ensures that the ordering of items in a slice is taken into account

`WithCombinedIdentifierJoinString(joinSep rune)` when using a combined identifier this character is used to join all identifiers to one string for representation in path

`WithSummarizeMissingStructs()` if a struct is added/removed on right this notes the difference as just one entry instead of every field on its own 

`WithStructMapKeys()` enables the possibility to use complex values as struct keys. It gets encoded with gob/base64 to prevent problems when building path and comparing by key

`WithEmbeddedStructsAsField()` if the struct has another struct embedded and this is set - the embedded struct will be listed as its own field with the struct fields as sub fields


## Differences

When you have Differences you can use them in your code and process them as needed. To make this easier there are some 
filter functions to only iterate over wanted Differences.

```go
    var diffs gompare.Differences
    ...
    for diff := range diffs.GetDifferences(cmp.WhereDiffType(dt), cmp.WherePathAt(attr, 0)) {
		...
    }
```

### Supported filters

`WherePath(p string)` - checks if one path segments matches with `p`

`WherePathAt(p string, idx int)` - checks if the path segment at index `idx` matches `p`

`WherePathDepth(l int)` - checks if the path length is equal `l`

`WherePathDepthGt(l int)` - checks if the path length is greater than `l`

`WherePathDepthLt(l int)` - checks if the path length is less than `l`

`WhereDiffType(dt DiffType)` - checks if the Difference type isequal to `dt`

`WhereOr(filterFunc ...DifferenceFilterFunc)` - if you add multiple filter to `GetDifferences()` those filter are logical AND - this functions combine than logical OR. With this you can build almost any filter

## Running Tests

```
go test -v
```

## Versioning

For transparency into our release cycle and in striving to maintain backward
compatibility, this project is maintained under [the Semantic Versioning guidelines](http://semver.org/).
