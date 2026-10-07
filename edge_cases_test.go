package gompare

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type idedElem struct {
	ID    string `cmp:"id,identifier"`
	Count int    `cmp:"count"`
}

type brokenTemplateElem struct {
	A int `cmp:"a,identifier:{{.A"`
	B int `cmp:"b,identifier:{{.A"`
}

func TestOptionsValidation(t *testing.T) {
	if _, err := NewComparer(WithTagName("")); !errors.Is(err, ErrInvalidOption) {
		t.Errorf("expected ErrInvalidOption, got %v", err)
	}
	if _, err := Compare(1, 2, WithTagName("")); !errors.Is(err, ErrInvalidOption) {
		t.Errorf("expected ErrInvalidOption from Compare, got %v", err)
	}
}

func TestMissingScalarsAndInterfaces(t *testing.T) {
	diffs, err := Compare(map[string]float64{}, map[string]float64{"a": 1.5})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: ADDED, Path: []string{"a"}, Right: 1.5}})

	// interface content removed: the slice inside is listed by element
	diffs, err = Compare(map[string]any{"a": []int{1}}, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: REMOVED, Path: []string{"a", "0"}, Left: 1}})

	// a nil interface that is missing on the other side is one entry
	diffs, err = Compare(map[string]any{}, map[string]any{"a": nil})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: ADDED, Path: []string{"a"}, Right: nil}})

	// unsupported kinds behind a missing side name the present kind
	_, err = Compare(nil, func() {})
	if !errors.Is(err, ErrUnsupportedType) || !strings.Contains(err.Error(), "func") {
		t.Errorf("expected ErrUnsupportedType naming func, got %v", err)
	}
	_, err = Compare(nil, map[string]any{"f": func() {}})
	if !errors.Is(err, ErrUnsupportedType) {
		t.Errorf("expected ErrUnsupportedType from a missing map value, got %v", err)
	}
	_, err = Compare(nil, struct{ F func() }{})
	if !errors.Is(err, ErrUnsupportedType) {
		t.Errorf("expected ErrUnsupportedType from a missing struct field, got %v", err)
	}
}

func TestSummarizeMissingRemoved(t *testing.T) {
	type holder struct {
		S []int
		M map[string]int
	}
	diffs, err := Compare(holder{S: []int{1}, M: map[string]int{"k": 1}}, holder{}, WithSummarizeMissing())
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: REMOVED, Path: []string{"S"}, Left: []int{1}},
		{Type: REMOVED, Path: []string{"M"}, Left: map[string]int{"k": 1}},
	})

	diffs, err = Compare(SimpleStructNoTag{Name: "a"}, nil, WithSummarizeMissingStructs())
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: REMOVED, Path: []string{}, Left: SimpleStructNoTag{Name: "a"}}})
}

func TestIdentifierErrorsOnEitherSide(t *testing.T) {
	// isComparable decides on the first struct, later elements with a broken template fail on their side
	_, err := Compare([]any{idedElem{"a", 1}, brokenTemplateElem{1, 2}}, []any{idedElem{"a", 1}})
	if !errors.Is(err, ErrIdentifierTemplate) || !strings.Contains(err.Error(), "path") {
		t.Errorf("expected ErrIdentifierTemplate on the left side, got %v", err)
	}
	_, err = Compare([]any{idedElem{"a", 1}}, []any{idedElem{"a", 1}, brokenTemplateElem{1, 2}})
	if !errors.Is(err, ErrIdentifierTemplate) {
		t.Errorf("expected ErrIdentifierTemplate on the right side, got %v", err)
	}
}

func TestTagOptionsAreScanned(t *testing.T) {
	type tagged struct {
		Name string `cmp:"name,other,identifier"`
		V    int    `cmp:"v,other"`
	}
	diffs, err := Compare([]tagged{{"a", 1}}, []tagged{{"a", 2}})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"a", "v"}, Left: 1, Right: 2}})

	type untagged struct {
		Name string `cmp:"name,other"`
	}
	diffs, err = Compare([]untagged{{"a"}}, []untagged{{"b"}})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"0", "name"}, Left: "a", Right: "b"}})
}

func TestMapKeyKindsAndEncodingErrors(t *testing.T) {
	diffs, err := Compare(map[uint8]int{10: 1, 2: 1}, map[uint8]int{10: 2, 2: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 2 || diffs[0].Path[0] != "2" || diffs[1].Path[0] != "10" {
		t.Errorf("uint keys should sort numerically, got %+v", diffs)
	}

	diffs, err = Compare(map[float64]int{2.5: 1, 1.5: 1}, map[float64]int{2.5: 2, 1.5: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 2 || diffs[0].Path[0] != "1.5" || diffs[1].Path[0] != "2.5" {
		t.Errorf("float keys should sort by their text, got %+v", diffs)
	}

	// a key gob cannot encode (chan field) is an error with the option, printed without it
	type badKey struct{ C chan int }
	_, err = Compare(map[badKey]int{{}: 1}, map[badKey]int{{}: 2}, WithStructMapKeys())
	if err == nil || !strings.Contains(err.Error(), "cannot encode map key") {
		t.Errorf("expected a key encoding error, got %v", err)
	}
	_, err = Compare(nil, map[badKey]int{{}: 2}, WithStructMapKeys())
	if err == nil || !strings.Contains(err.Error(), "cannot encode map key") {
		t.Errorf("expected a key encoding error for a missing map, got %v", err)
	}
	diffs, err = Compare(map[badKey]int{{}: 1}, map[badKey]int{{}: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(diffs) != 1 {
		t.Errorf("expected one difference, got %+v", diffs)
	}
}

func TestUnorderedSliceOfStructsWithDuplicates(t *testing.T) {
	type plain struct{ A int }
	diffs, err := Compare([]plain{{1}, {1}}, []plain{{1}, {2}})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"1", "A"}, Left: 1, Right: 2}})
}

func TestOrderedSlicesOfDifferentLength(t *testing.T) {
	diffs, err := Compare([]int{1, 2, 3}, []int{1, 2}, WithSliceOrdering())
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: REMOVED, Path: []string{"2"}, Left: 3}})
}

func TestDifferenceFilters(t *testing.T) {
	diffs := Differences{
		{Type: ADDED, Path: []string{"a"}},
		{Type: CHANGED, Path: []string{"a", "b"}},
		{Type: REMOVED, Path: []string{"x", "b", "c"}},
	}
	collect := func(filters ...DifferenceFilterFunc) []string {
		var paths []string
		for d := range diffs.GetDifferences(filters...) {
			paths = append(paths, strings.Join(d.Path, "/"))
		}
		return paths
	}
	cases := []struct {
		name    string
		filters []DifferenceFilterFunc
		want    []string
	}{
		{"none", nil, []string{"a", "a/b", "x/b/c"}},
		{"path", []DifferenceFilterFunc{WherePath("b")}, []string{"a/b", "x/b/c"}},
		{"path-at", []DifferenceFilterFunc{WherePathAt("a", 0)}, []string{"a", "a/b"}},
		{"path-at-out-of-range", []DifferenceFilterFunc{WherePathAt("c", 2)}, []string{"x/b/c"}},
		{"depth", []DifferenceFilterFunc{WherePathDepth(2)}, []string{"a/b"}},
		{"depth-gt", []DifferenceFilterFunc{WherePathDepthGt(1)}, []string{"a/b", "x/b/c"}},
		{"depth-lt", []DifferenceFilterFunc{WherePathDepthLt(2)}, []string{"a"}},
		{"type", []DifferenceFilterFunc{WhereDiffType(REMOVED)}, []string{"x/b/c"}},
		{"and", []DifferenceFilterFunc{WherePath("b"), WhereDiffType(CHANGED)}, []string{"a/b"}},
		{"or", []DifferenceFilterFunc{WhereOr(WhereDiffType(ADDED), WhereDiffType(REMOVED))}, []string{"a", "x/b/c"}},
		{"or-empty", []DifferenceFilterFunc{WhereOr()}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := collect(tc.filters...); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, wanted %v", got, tc.want)
			}
		})
	}

	if !diffs.HasDifferences() || !diffs.HasDifferences(WherePath("c")) || diffs.HasDifferences(WherePath("nope")) {
		t.Error("HasDifferences gave a wrong answer")
	}

	// the iterator stops when the consumer breaks
	n := 0
	for range diffs.GetDifferences() {
		n++
		break
	}
	if n != 1 {
		t.Errorf("expected the iteration to stop after one element, got %d", n)
	}
}

// The second pass over an unordered slice can fail where the first one did not: with
// WithAllowDifferentStructs the left struct drives the field iteration, so a field that only
// exists on one side is only visited when that side is on the left.
func TestUnorderedSliceErrorOnSecondPass(t *testing.T) {
	type withFunc struct {
		X int
		F func()
	}
	type without struct{ X int }

	_, err := Compare([]any{withFunc{X: 1}}, []any{without{X: 1}}, WithAllowDifferentStructs())
	if !errors.Is(err, ErrUnsupportedType) {
		t.Errorf("expected ErrUnsupportedType from the second pass, got %v", err)
	}
}
