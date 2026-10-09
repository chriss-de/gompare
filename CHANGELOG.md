# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [2.2.0] - 2026-10-09

Updated Changelog.

## [2.1.0] - 2026-10-08

### Added
- An `identifier:<template>` option on a slice, array or map field identifies the elements of that field
  and overrides the identifier tags of the element type. The template sees every field of the element.
  `nil` elements fall back to their index, elements that are not structs return `ErrIdentifierTemplate`.
  On a map field the entries are matched and listed by the rendered identifier instead of the map key.
  A bare `identifier` on such a field returns `ErrIdentifierTemplate`.
- The `array_identifier` tag option uses an array field as a whole as (part of) the identifier of its struct.
  A bare `identifier` on an array field did that before and is now an error, see above.
  `array_identifier` on a field that is not an array returns `ErrInvalidOption`.
- Map entries that render to the same identifier return `ErrDuplicateIdentifier`, like slice elements do.

### Fixed
- A template on a single identifier field is used. Before it was ignored and the raw value was used.
- An identifier field holding a value that cannot be a map key returns `ErrIdentifierTemplate` instead of panicking.

## [2.0.0] - 2026-10-07

The module path is now `github.com/chriss-de/gompare/v2`.

### Removed
- `ErrInvalidChangeType` (was unreachable).
- The internal types `ComparableList`, `ComparableListEntry`, `NewComparableList`, `Type`, `CompareFunc`
  and the compare kind constants are no longer exported.

### Changed (behaviour)
- `Left` and `Right` always hold the value in the type of its field. Values behind unexported fields were `int64`/`uint64`/`float64` before.
- Missing containers are listed by their content everywhere: at the top level, behind `nil` pointers and interfaces,
  and inside missing structs. A scalar behind a `nil` pointer is one `changed` entry with the dereferenced value.
- Map keys are encoded the same way in every code path: scalars as text, other keys with `fmt.Sprint` or,
  with `WithStructMapKeys()`, as base64 encoded gob.
- Structs of different types are a type mismatch (`ErrTypeMismatch`) unless `WithAllowDifferentStructs()` is set.
- Added or removed structs now report every field, including fields holding their zero value.
- Structs reached through unexported fields are compared. Before they were skipped silently.
- Slice elements without an identifier (e.g. `nil` pointers) are matched by their index instead of being dropped.
- Map differences are emitted in sorted key order (integers numerically) instead of random map order.
- Errors are wrapped with the path where they happened. Match them with `errors.Is`.
- A missing `time.Time` is reported as one entry instead of its internal fields.
- `Compare(nil, nil)` returns no error.
- Identifier templates that reference an unknown key return `ErrIdentifierTemplate` instead of rendering `<no value>`.
- Identifier template data holds every identifier value under its Go field name and under its tag name.
- Errors raised while probing elements of an unordered slice propagate. A type mismatch between elements still means "not equal".

### Added
- `WithSummarizeMissing()` reports a missing struct, slice, array or map as one entry.
- `WithAllowDifferentStructs()` compares structs of different types field by field.
- `WithAllowTypeMismatch()` reports values of different kind as `changed` instead of returning `ErrTypeMismatch`.
- `ErrDuplicateIdentifier`, `ErrIdentifierTemplate`, `ErrUnsupportedType`, `ErrUnexportedField`, `ErrInvalidOption`.
- `WithTagName("")` is rejected with `ErrInvalidOption`.
- Runnable examples (`ExampleCompare`, `ExampleDifferences_GetDifferences`) whose output is verified by `go test`.
- Option functions return the named `CompareOptsFunc` type.
- Identifier templates may contain `:`.
- Benchmarks and a CI workflow (gofmt, vet, race tests, staticcheck).

### Fixed
- One `Comparer` can be used from multiple goroutines. Each call has its own state.
- Cycles through pointers, maps and slices terminate instead of recursing forever.
- A missing map no longer rewrites differences of sibling keys that share a path prefix.
- Duplicate identifiers within one slice return `ErrDuplicateIdentifier` instead of silently overwriting each other.
- Identifier template errors return an error instead of panicking.
- Identifier matching is used even if the first slice element is a `nil` pointer.
- Unordered comparison of slices with basic element kinds uses an index instead of a quadratic scan.
- The unsafe access to unexported fields is verified at compile time and at startup.

## [1.1.0] and earlier
See the git history.

[2.2.0]: https://github.com/chriss-de/gompare/compare/v2.1.0...v2.2.0
[2.1.0]: https://github.com/chriss-de/gompare/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/chriss-de/gompare/compare/v1.1.0...v2.0.0
[1.1.0]: https://github.com/chriss-de/gompare/releases/tag/v1.1.0
