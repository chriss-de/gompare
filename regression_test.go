package gompare

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// assertDiffs compares got and want ignoring order (map iteration order is random)
func assertDiffs(t *testing.T, got, want Differences) {
	t.Helper()
	key := func(d Difference) string { return string(d.Type) + ":" + strings.Join(d.Path, "/") }
	sort.Slice(got, func(i, j int) bool { return key(got[i]) < key(got[j]) })
	sort.Slice(want, func(i, j int) bool { return key(want[i]) < key(want[j]) })
	if len(got) != len(want) {
		t.Fatalf("unexpected number of differences - got: %d, wanted: %d\n got:  %+v\n want: %+v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i].Type != want[i].Type || key(got[i]) != key(want[i]) ||
			!reflect.DeepEqual(got[i].Left, want[i].Left) || !reflect.DeepEqual(got[i].Right, want[i].Right) {
			t.Errorf("difference %d - got: %+v, wanted: %+v", i, got[i], want[i])
		}
	}
}

// Bug: a Comparer reused from several goroutines shares one result slice.
// Run with -race to see the data race; without -race the results get mixed up.
func TestComparerConcurrentReuse(t *testing.T) {
	type item struct{ A int }
	c, err := NewComparer(WithSliceOrdering())
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for n := 0; n < 50; n++ {
				left := make([]item, 20)
				right := make([]item, 20)
				for i := range left {
					left[i] = item{A: i}
					right[i] = item{A: i + 1000*(g+1)}
				}
				diffs, err := c.Compare(left, right)
				if err != nil {
					t.Errorf("goroutine %d: unexpected error: %v", g, err)
					return
				}
				if len(diffs) != 20 {
					t.Errorf("goroutine %d: got %d differences, wanted 20", g, len(diffs))
					return
				}
				for _, d := range diffs {
					if d.Right.(int)-d.Left.(int) != 1000*(g+1) {
						t.Errorf("goroutine %d: got difference from another goroutine: %+v", g, d)
						return
					}
				}
			}
		}(g)
	}
	wg.Wait()
}

// Bug: self referencing pointers recurse forever
func TestCyclicPointers(t *testing.T) {
	type node struct {
		V    int
		Next *node
	}
	run := func(left, right *node) (Differences, error) {
		type result struct {
			d   Differences
			err error
		}
		ch := make(chan result, 1)
		go func() {
			d, err := Compare(left, right)
			ch <- result{d, err}
		}()
		select {
		case r := <-ch:
			return r.d, r.err
		case <-time.After(5 * time.Second):
			t.Fatal("Compare did not terminate on cyclic pointers")
			return nil, nil
		}
	}

	a := &node{V: 1}
	a.Next = a
	b := &node{V: 1}
	b.Next = b
	diffs, err := run(a, b)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{})

	c := &node{V: 2}
	c.Next = c
	diffs, err = run(a, c)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"V"}, Left: 1, Right: 2}})

	// a cycle through a slice and an interface
	type tree struct {
		Name     string
		Parent   *tree
		Children []*tree
		Any      any
	}
	rootA := &tree{Name: "root"}
	rootA.Children = []*tree{{Name: "child", Parent: rootA}}
	rootA.Any = rootA
	rootB := &tree{Name: "root"}
	rootB.Children = []*tree{{Name: "child2", Parent: rootB}}
	rootB.Any = rootB

	ch := make(chan Differences, 1)
	go func() {
		d, _ := Compare(rootA, rootB)
		ch <- d
	}()
	select {
	case d := <-ch:
		if !d.HasDifferences(WherePath("Name")) {
			t.Errorf("expected a Name difference, got %+v", d)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Compare did not terminate on cyclic tree")
	}
}

// Bug: when a map value is missing on one side, patching the result used a string prefix
// match on the path and also rewrote differences of sibling keys like "Mab" vs "Ma"
func TestMapMissingDoesNotPatchSiblings(t *testing.T) {
	left := map[string]map[string]int{"Mab": {"k": 1}}
	right := map[string]map[string]int{"Mab": {"k": 2}, "Ma": {"x": 1}}

	diffs, err := Compare(left, right)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: CHANGED, Path: []string{"Mab", "k"}, Left: 1, Right: 2},
		{Type: ADDED, Path: []string{"Ma", "x"}, Left: nil, Right: 1},
	})

	diffs, err = Compare(right, left)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: CHANGED, Path: []string{"Mab", "k"}, Left: 2, Right: 1},
		{Type: REMOVED, Path: []string{"Ma", "x"}, Left: 1, Right: nil},
	})
}

// Bug: a time.Time that is missing on one side was treated as a plain struct
// and reported by its internal fields wall/ext/loc
func TestTimeMissingOnOneSide(t *testing.T) {
	diffs, err := Compare(map[string]time.Time{}, map[string]time.Time{"a": testTimeA})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: ADDED, Path: []string{"a"}, Left: nil, Right: testTimeA}})

	diffs, err = Compare(map[string]time.Time{"a": testTimeA}, map[string]time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: REMOVED, Path: []string{"a"}, Left: testTimeA, Right: nil}})

	diffs, err = Compare(nil, testTimeA)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: ADDED, Path: []string{}, Left: nil, Right: testTimeA}})
}

// Bug: comparing nil with nil returned ErrTypeMismatch
func TestNilWithNil(t *testing.T) {
	diffs, err := Compare(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertDiffs(t, diffs, Differences{})
}

// Bug: a changed dynamic type inside an interface aborted the whole comparison.
// The default stays an error, WithAllowTypeMismatch reports it as CHANGED.
func TestAllowTypeMismatch(t *testing.T) {
	left := map[string]any{"a": 1, "b": "same"}
	right := map[string]any{"a": "x", "b": "same"}

	_, err := Compare(left, right)
	if !errors.Is(err, ErrTypeMismatch) {
		t.Fatalf("expected ErrTypeMismatch by default, got %v", err)
	}

	diffs, err := Compare(left, right, WithAllowTypeMismatch())
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"a"}, Left: 1, Right: "x"}})

	_, err = Compare(1, "x")
	if !errors.Is(err, ErrTypeMismatch) {
		t.Fatalf("expected ErrTypeMismatch by default, got %v", err)
	}
	diffs, err = Compare(1, "x", WithAllowTypeMismatch())
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{}, Left: 1, Right: "x"}})
}

// assertDiffsLoose is like assertDiffs but compares Left/Right by their printed value.
// Values reached through unexported fields come back as int64 instead of int.
func assertDiffsLoose(t *testing.T, got, want Differences) {
	t.Helper()
	key := func(d Difference) string { return string(d.Type) + ":" + strings.Join(d.Path, "/") }
	sort.Slice(got, func(i, j int) bool { return key(got[i]) < key(got[j]) })
	sort.Slice(want, func(i, j int) bool { return key(want[i]) < key(want[j]) })
	if len(got) != len(want) {
		t.Fatalf("unexpected number of differences - got: %d, wanted: %d\n got:  %+v\n want: %+v", len(got), len(want), got, want)
	}
	for i := range got {
		if key(got[i]) != key(want[i]) || fmt.Sprint(got[i].Left) != fmt.Sprint(want[i].Left) || fmt.Sprint(got[i].Right) != fmt.Sprint(want[i].Right) {
			t.Errorf("difference %d - got: %+v, wanted: %+v", i, got[i], want[i])
		}
	}
}

// Bug: fields with a zero value were not reported when the whole struct was added or removed
func TestMissingStructReportsZeroFields(t *testing.T) {
	type inner struct{ A int }
	type removable struct {
		Name  string
		Count int
		On    bool
		T     time.Time
		Inner inner
		Items []int
		M     map[string]int
		P     *inner
	}

	val := removable{Name: "n", Items: []int{0}, M: map[string]int{"k": 0}}

	diffs, err := Compare(val, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: REMOVED, Path: []string{"Name"}, Left: "n"},
		{Type: REMOVED, Path: []string{"Count"}, Left: 0},
		{Type: REMOVED, Path: []string{"On"}, Left: false},
		{Type: REMOVED, Path: []string{"T"}, Left: time.Time{}},
		{Type: REMOVED, Path: []string{"Inner", "A"}, Left: 0},
		{Type: REMOVED, Path: []string{"Items", "0"}, Left: 0},
		{Type: REMOVED, Path: []string{"M", "k"}, Left: 0},
	})

	diffs, err = Compare(nil, val)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: ADDED, Path: []string{"Name"}, Right: "n"},
		{Type: ADDED, Path: []string{"Count"}, Right: 0},
		{Type: ADDED, Path: []string{"On"}, Right: false},
		{Type: ADDED, Path: []string{"T"}, Right: time.Time{}},
		{Type: ADDED, Path: []string{"Inner", "A"}, Right: 0},
		{Type: ADDED, Path: []string{"Items", "0"}, Right: 0},
		{Type: ADDED, Path: []string{"M", "k"}, Right: 0},
	})

	// also when the struct is an element of an identifiable slice
	type ided struct {
		ID    string `cmp:"id,identifier"`
		Count int    `cmp:"count"`
	}
	diffs, err = Compare([]ided{}, []ided{{ID: "x", Count: 0}})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: ADDED, Path: []string{"x", "id"}, Right: "x"},
		{Type: ADDED, Path: []string{"x", "count"}, Right: 0},
	})
}

// Bug: structs reached through an unexported field were skipped entirely,
// while unexported scalar fields were compared
func TestUnexportedNestedStruct(t *testing.T) {
	type ided struct {
		ID    string `cmp:"id,identifier"`
		Count int    `cmp:"count"`
	}
	type inner struct {
		a     int
		Items []ided
		M     map[string]int
		T     time.Time
	}
	type outer struct {
		in inner
		x  int
	}

	left := outer{in: inner{a: 1, Items: []ided{{"k", 1}}, M: map[string]int{"m": 1}, T: testTimeA}, x: 1}
	right := outer{in: inner{a: 2, Items: []ided{{"k", 2}, {"n", 0}}, M: map[string]int{"m": 2}, T: testTimeB}, x: 2}

	diffs, err := Compare(left, right)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffsLoose(t, diffs, Differences{
		{Type: CHANGED, Path: []string{"in", "a"}, Left: 1, Right: 2},
		{Type: CHANGED, Path: []string{"in", "Items", "k", "count"}, Left: 1, Right: 2},
		{Type: ADDED, Path: []string{"in", "Items", "n", "id"}, Right: "n"},
		{Type: ADDED, Path: []string{"in", "Items", "n", "count"}, Right: 0},
		{Type: CHANGED, Path: []string{"in", "M", "m"}, Left: 1, Right: 2},
		{Type: CHANGED, Path: []string{"in", "T"}, Left: testTimeA, Right: testTimeB},
		{Type: CHANGED, Path: []string{"x"}, Left: 1, Right: 2},
	})

	// struct removed behind an unexported field
	diffs, err = Compare(left, outer{})
	if err != nil {
		t.Fatal(err)
	}
	if !diffs.HasDifferences(WherePathAt("in", 0), WherePathAt("a", 1)) {
		t.Errorf("expected a difference for in/a, got %+v", diffs)
	}
}

// Bug: elements of an identifiable slice that have no identifier (e.g. nil pointers) were dropped silently
func TestIdentifiableSliceElementWithoutIdentifier(t *testing.T) {
	type ided struct {
		ID    string `cmp:"id,identifier"`
		Count int    `cmp:"count"`
	}

	diffs, err := Compare([]*ided{{"a", 1}, nil}, []*ided{{"a", 1}})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: REMOVED, Path: []string{"1"}, Left: (*ided)(nil), Right: nil},
	})

	diffs, err = Compare([]*ided{{"a", 1}}, []*ided{{"a", 1}, nil})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: ADDED, Path: []string{"1"}, Left: nil, Right: (*ided)(nil)},
	})

	// elements without identifier in a mixed slice fall back to their index, identifiers are untouched
	diffs, err = Compare([]any{ided{"a", 1}, 5}, []any{ided{"a", 2}, 6})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: CHANGED, Path: []string{"a", "count"}, Left: 1, Right: 2},
		{Type: CHANGED, Path: []string{"1"}, Left: 5, Right: 6},
	})
}

// Bug: duplicate identifiers within one slice silently overwrote each other
func TestDuplicateIdentifier(t *testing.T) {
	type ided struct {
		ID    string `cmp:"id,identifier"`
		Count int    `cmp:"count"`
	}

	_, err := Compare([]ided{{"a", 1}, {"a", 2}}, []ided{{"a", 3}})
	if !errors.Is(err, ErrDuplicateIdentifier) {
		t.Fatalf("expected ErrDuplicateIdentifier on left, got %v", err)
	}

	_, err = Compare([]ided{{"a", 3}}, []ided{{"a", 1}, {"a", 2}})
	if !errors.Is(err, ErrDuplicateIdentifier) {
		t.Fatalf("expected ErrDuplicateIdentifier on right, got %v", err)
	}

	_, err = Compare(struct{ L []ided }{[]ided{{"a", 1}, {"a", 2}}}, struct{ L []ided }{})
	if !errors.Is(err, ErrDuplicateIdentifier) {
		t.Fatalf("expected ErrDuplicateIdentifier, got %v", err)
	}
	if !strings.Contains(err.Error(), "L") || !strings.Contains(err.Error(), "a") {
		t.Errorf("error should name path and identifier, got: %v", err)
	}
}

// Bug: map Differences came out in random map iteration order, which made results
// (and the mixed-slice-map test case) non deterministic
func TestMapDifferencesAreSorted(t *testing.T) {
	left := map[string]int{}
	right := map[string]int{}
	var want Differences
	for _, k := range []string{"j", "c", "a", "h", "b", "f", "e", "i", "d", "g"} {
		left[k] = 1
		right[k] = 2
	}
	for _, k := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		want = append(want, Difference{Type: CHANGED, Path: []string{k}, Left: 1, Right: 2})
	}

	for n := 0; n < 20; n++ {
		diffs, err := Compare(left, right)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(diffs, want) {
			t.Fatalf("run %d: differences not sorted by key:\n got:  %+v\n want: %+v", n, diffs, want)
		}

		diffs, err = Compare(nil, right)
		if err != nil {
			t.Fatal(err)
		}
		for i := range diffs {
			if i > 0 && diffs[i-1].Path[0] > diffs[i].Path[0] {
				t.Fatalf("run %d: added map differences not sorted by key: %+v", n, diffs)
			}
		}
	}

	// integer keys sort numerically, not as strings
	diffs, err := Compare(map[int]int{10: 1, 2: 1, 1: 1}, map[int]int{10: 2, 2: 2, 1: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 3 || diffs[0].Path[0] != "1" || diffs[1].Path[0] != "2" || diffs[2].Path[0] != "10" {
		t.Errorf("integer keys not sorted numerically: %+v", diffs)
	}
	assertDiffs(t, diffs, Differences{
		{Type: CHANGED, Path: []string{"1"}, Left: 1, Right: 2},
		{Type: CHANGED, Path: []string{"2"}, Left: 1, Right: 2},
		{Type: CHANGED, Path: []string{"10"}, Left: 1, Right: 2},
	})
}

// noPanic runs f and converts a panic into a test failure instead of aborting the test binary
func noPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("unexpected panic: %v", p)
		}
	}()
	f()
}

// Bug: identifier templates panicked on errors and silently rendered unknown keys as "<no value>"
func TestIdentifierTemplateErrors(t *testing.T) {
	type broken struct {
		A int `cmp:"a,identifier:{{.A"`
		B int `cmp:"b,identifier:{{.A"`
	}
	type mismatched struct {
		A int `cmp:"a,identifier:{{.A}}"`
		B int `cmp:"b,identifier:{{.B}}"`
	}
	type unknownKey struct {
		A int `cmp:"a,identifier:{{.A}}-{{.nope}}"`
		B int `cmp:"b,identifier:{{.A}}-{{.nope}}"`
	}
	type tagNames struct {
		Name string `cmp:"name,identifier:{{.name}}-{{.ID}}"`
		ID   int    `cmp:"id,identifier:{{.name}}-{{.ID}}"`
		V    int    `cmp:"v"`
	}

	noPanic(t, func() {
		_, err := Compare([]broken{{1, 2}}, []broken{{1, 3}})
		if !errors.Is(err, ErrIdentifierTemplate) {
			t.Errorf("broken template: expected ErrIdentifierTemplate, got %v", err)
		}
	})
	noPanic(t, func() {
		_, err := Compare([]mismatched{{1, 2}}, []mismatched{{1, 3}})
		if !errors.Is(err, ErrIdentifierTemplate) {
			t.Errorf("mismatched templates: expected ErrIdentifierTemplate, got %v", err)
		}
	})
	noPanic(t, func() {
		_, err := Compare([]unknownKey{{1, 2}}, []unknownKey{{1, 3}})
		if !errors.Is(err, ErrIdentifierTemplate) {
			t.Errorf("unknown key: expected ErrIdentifierTemplate, got %v", err)
		}
		if err != nil && !strings.Contains(err.Error(), "nope") {
			t.Errorf("error should name the unknown key, got: %v", err)
		}
	})

	// tag names work as template keys as well as Go field names
	diffs, err := Compare([]tagNames{{"x", 1, 0}}, []tagNames{{"x", 1, 5}})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"x-1", "v"}, Left: 0, Right: 5}})
}

// Bug: a ':' inside an identifier template cut the template off and it was silently ignored
func TestIdentifierTemplateWithColon(t *testing.T) {
	type colon struct {
		A int `cmp:"a,identifier:{{.A}}:{{.B}}"`
		B int `cmp:"b,identifier:{{.A}}:{{.B}}"`
		V int `cmp:"v"`
	}
	diffs, err := Compare([]colon{{1, 2, 0}}, []colon{{1, 2, 1}})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"1:2", "v"}, Left: 0, Right: 1}})
}

// Bug: errors did not tell where in the structure they happened
func TestErrorsCarryPath(t *testing.T) {
	_, err := Compare(map[string]map[string]any{"x": {"y": 1}}, map[string]map[string]any{"x": {"y": "s"}})
	if !errors.Is(err, ErrTypeMismatch) {
		t.Fatalf("expected ErrTypeMismatch, got %v", err)
	}
	if !strings.Contains(err.Error(), "x/y") {
		t.Errorf("type mismatch error should contain the path x/y, got: %v", err)
	}

	type withFunc struct {
		Outer struct{ F func() }
	}
	_, err = Compare(withFunc{}, withFunc{})
	if !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("expected ErrUnsupportedType, got %v", err)
	}
	if !strings.Contains(err.Error(), "Outer/F") || !strings.Contains(err.Error(), "func") {
		t.Errorf("unsupported type error should contain path and kind, got: %v", err)
	}

	type ided struct {
		ID string `cmp:"id,identifier"`
	}
	_, err = Compare(struct{ L []ided }{[]ided{{"a"}, {"a"}}}, struct{ L []ided }{})
	if !errors.Is(err, ErrDuplicateIdentifier) || !strings.Contains(err.Error(), `"L"`) {
		t.Errorf("duplicate identifier error should contain the path, got: %v", err)
	}
}

// Bug: errors other than a type mismatch raised while probing elements of an unordered slice were swallowed
func TestSliceTrackerPropagatesErrors(t *testing.T) {
	type broken struct {
		A int `cmp:"a,identifier:{{.A"`
		B int `cmp:"b,identifier:{{.A"`
	}
	type elem struct {
		Items []broken
	}
	noPanic(t, func() {
		_, err := Compare([]elem{{[]broken{{1, 2}}}}, []elem{{[]broken{{1, 3}}}})
		if !errors.Is(err, ErrIdentifierTemplate) {
			t.Errorf("expected ErrIdentifierTemplate, got %v", err)
		}
	})

	// a type mismatch between elements of a []any still just means "not equal"
	diffs, err := Compare([]any{1, "a"}, []any{"a", 1})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{})
}

// Hardening: access to unexported fields relies on the memory layout of reflect.Value.
// The package verifies that at startup; if it ever stops working Compare returns an error
// instead of reading garbage.
func TestUnexportedAccessGuard(t *testing.T) {
	if !checkUnexportedAccess() {
		t.Fatal("unexported field access self test failed on this Go version")
	}

	type hidden struct{ a int }
	diffs, err := Compare(hidden{1}, hidden{2})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffsLoose(t, diffs, Differences{{Type: CHANGED, Path: []string{"a"}, Left: 1, Right: 2}})

	unexportedAccessOK = false
	defer func() { unexportedAccessOK = true }()

	_, err = Compare(hidden{1}, hidden{2})
	if !errors.Is(err, ErrUnexportedField) || !strings.Contains(err.Error(), `"a"`) {
		t.Errorf("expected ErrUnexportedField naming the path, got %v", err)
	}
	// exported fields keep working
	diffs, err = Compare(struct{ A int }{1}, struct{ A int }{2})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"A"}, Left: 1, Right: 2}})
}
