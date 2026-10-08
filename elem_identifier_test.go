package gompare

import (
	"errors"
	"strings"
	"testing"
)

// grant has its own identifier, right carries none
type grant struct {
	Code  string `cmp:"code,identifier"`
	Level int    `cmp:"level"`
}

type right struct {
	Name   string  `cmp:"name"`
	Scope  string  `cmp:"scope"`
	Grants []grant `cmp:"grants"`
}

// Feature: an identifier template on a slice, array or map field identifies the elements of that field
func TestElemIdentifierOnSlice(t *testing.T) {
	type user struct {
		Rights []right `cmp:"rights,identifier:{{.name}}@{{.Scope}}"`
	}

	left := user{Rights: []right{{Name: "read", Scope: "all"}, {Name: "write", Scope: "own"}}}
	right_ := user{Rights: []right{{Name: "write", Scope: "own", Grants: []grant{{"g", 1}}}, {Name: "read", Scope: "team"}}}

	diffs, err := Compare(left, right_)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: REMOVED, Path: []string{"rights", "read@all", "name"}, Left: "read"},
		{Type: REMOVED, Path: []string{"rights", "read@all", "scope"}, Left: "all"},
		{Type: ADDED, Path: []string{"rights", "write@own", "grants", "g", "code"}, Right: "g"},
		{Type: ADDED, Path: []string{"rights", "write@own", "grants", "g", "level"}, Right: 1},
		{Type: ADDED, Path: []string{"rights", "read@team", "name"}, Right: "read"},
		{Type: ADDED, Path: []string{"rights", "read@team", "scope"}, Right: "team"},
	})
}

// the override wins over the identifier tags of the element type
func TestElemIdentifierOverridesElementTags(t *testing.T) {
	type user struct {
		Grants []grant `cmp:"grants,identifier:{{.code}}/{{.level}}"`
	}

	diffs, err := Compare(user{Grants: []grant{{"g", 1}}}, user{Grants: []grant{{"g", 2}}})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: REMOVED, Path: []string{"grants", "g/1", "code"}, Left: "g"},
		{Type: REMOVED, Path: []string{"grants", "g/1", "level"}, Left: 1},
		{Type: ADDED, Path: []string{"grants", "g/2", "code"}, Right: "g"},
		{Type: ADDED, Path: []string{"grants", "g/2", "level"}, Right: 2},
	})
}

// the override applies one level down only: nested slices keep their own identifiers
func TestElemIdentifierAppliesOneLevelDown(t *testing.T) {
	type withGrants struct {
		Rights []right `cmp:"rights,identifier:{{.Name}}"`
	}
	type twoLevels struct {
		Users []withGrants `cmp:"users,identifier:{{.Rights | len}}"`
	}

	left := twoLevels{Users: []withGrants{{Rights: []right{{Name: "r", Grants: []grant{{"g", 1}}}}}}}
	right_ := twoLevels{Users: []withGrants{{Rights: []right{{Name: "r", Grants: []grant{{"g", 2}}}}}}}

	diffs, err := Compare(left, right_)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: CHANGED, Path: []string{"users", "1", "rights", "r", "grants", "g", "level"}, Left: 1, Right: 2},
	})

	// a sibling slice is not affected either, even if the override field is a nil pointer on both sides
	type siblings struct {
		A *[]right `cmp:"a,identifier:{{.Name}}"`
		B []grant  `cmp:"b"`
		C []right  `cmp:"c"`
	}
	diffs, err = Compare(
		siblings{B: []grant{{"g", 1}}, C: []right{{Name: "x"}}},
		siblings{B: []grant{{"g", 2}}, C: []right{{Name: "y"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: CHANGED, Path: []string{"b", "g", "level"}, Left: 1, Right: 2},
		{Type: CHANGED, Path: []string{"c", "0", "name"}, Left: "x", Right: "y"},
	})
}

// the override reaches the container through pointers and interfaces and is used for missing containers
func TestElemIdentifierBehindPointerAndMissing(t *testing.T) {
	type user struct {
		Rights *[]right `cmp:"rights,identifier:{{.Name}}"`
	}
	type holder struct {
		U *user `cmp:"u"`
	}

	rs := []right{{Name: "r", Scope: "s"}}
	diffs, err := Compare(user{}, user{Rights: &rs})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: ADDED, Path: []string{"rights", "r", "name"}, Right: "r"},
		{Type: ADDED, Path: []string{"rights", "r", "scope"}, Right: "s"},
	})

	// the whole struct holding the slice is missing
	diffs, err = Compare(holder{U: &user{Rights: &rs}}, holder{})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: REMOVED, Path: []string{"u", "rights", "r", "name"}, Left: "r"},
		{Type: REMOVED, Path: []string{"u", "rights", "r", "scope"}, Left: "s"},
	})

	// summarized missing containers are unaffected
	diffs, err = Compare(user{}, user{Rights: &rs}, WithSummarizeMissing())
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: ADDED, Path: []string{"rights"}, Right: rs}})

	// nil elements fall back to their index
	type ptrs struct {
		Rights []*right `cmp:"rights,identifier:{{.Name}}"`
	}
	diffs, err = Compare(ptrs{Rights: []*right{nil, {Name: "r"}}}, ptrs{Rights: []*right{{Name: "r", Scope: "s"}}})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: REMOVED, Path: []string{"rights", "0"}, Left: (*right)(nil)},
		{Type: CHANGED, Path: []string{"rights", "r", "scope"}, Left: "", Right: "s"},
	})
}

func TestElemIdentifierOnArrayAndMap(t *testing.T) {
	type arr struct {
		Rights [2]right `cmp:"rights,identifier:{{.Name}}"`
	}
	diffs, err := Compare(
		arr{Rights: [2]right{{Name: "a"}, {Name: "b"}}},
		arr{Rights: [2]right{{Name: "b", Scope: "s"}, {Name: "a"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"rights", "b", "scope"}, Left: "", Right: "s"}})

	// map entries are matched and listed by the rendered identifier instead of the map key
	type m struct {
		Rights map[string]right `cmp:"rights,identifier:{{.Name}}"`
	}
	diffs, err = Compare(
		m{Rights: map[string]right{"k1": {Name: "a"}, "k2": {Name: "b"}}},
		m{Rights: map[string]right{"k9": {Name: "a", Scope: "s"}, "k2": {Name: "c"}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffsLoose(t, diffs, Differences{
		{Type: CHANGED, Path: []string{"rights", "a", "scope"}, Left: "", Right: "s"},
		{Type: REMOVED, Path: []string{"rights", "b", "name"}, Left: "b"},
		{Type: REMOVED, Path: []string{"rights", "b", "scope"}, Left: ""},
		{Type: ADDED, Path: []string{"rights", "c", "name"}, Right: "c"},
		{Type: ADDED, Path: []string{"rights", "c", "scope"}, Right: ""},
	})

	// a missing map lists its entries by identifier as well
	diffs, err = Compare(m{}, m{Rights: map[string]right{"k": {Name: "a"}}})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{
		{Type: ADDED, Path: []string{"rights", "a", "name"}, Right: "a"},
		{Type: ADDED, Path: []string{"rights", "a", "scope"}, Right: ""},
	})

	_, err = Compare(m{Rights: map[string]right{"k1": {Name: "a"}, "k2": {Name: "a"}}}, m{})
	if !errors.Is(err, ErrDuplicateIdentifier) {
		t.Errorf("expected ErrDuplicateIdentifier for a map, got %v", err)
	}
}

func TestElemIdentifierErrors(t *testing.T) {
	type bareSlice struct {
		Rights []right `cmp:"rights,identifier"`
	}
	type bareMap struct {
		Rights map[string]right `cmp:"rights,identifier"`
	}
	type bareArray struct {
		Rights *[1]right `cmp:"rights,identifier"`
	}
	type broken struct {
		Rights []right `cmp:"rights,identifier:{{.Name"`
	}
	type unknownKey struct {
		Rights []right `cmp:"rights,identifier:{{.nope}}"`
	}
	type notStruct struct {
		Names []string `cmp:"names,identifier:{{.}}"`
	}
	type dup struct {
		Rights []right `cmp:"rights,identifier:{{.Name}}"`
	}

	for name, fn := range map[string]func() error{
		"bare slice":        func() error { _, err := Compare(bareSlice{}, bareSlice{}); return err },
		"bare map":          func() error { _, err := Compare(bareMap{}, bareMap{}); return err },
		"bare array":        func() error { _, err := Compare(bareArray{}, bareArray{}); return err },
		"bare missing side": func() error { _, err := Compare(nil, bareSlice{}); return err },
		"bare as element":   func() error { _, err := Compare([]bareSlice{{}}, []bareSlice{{}}); return err },
		"broken template": func() error {
			_, err := Compare(broken{Rights: []right{{}}}, broken{})
			return err
		},
		"unknown key": func() error {
			_, err := Compare(unknownKey{Rights: []right{{}}}, unknownKey{})
			return err
		},
		"not a struct": func() error {
			_, err := Compare(notStruct{Names: []string{"a"}}, notStruct{})
			return err
		},
	} {
		noPanic(t, func() {
			err := fn()
			if !errors.Is(err, ErrIdentifierTemplate) {
				t.Errorf("%s: expected ErrIdentifierTemplate, got %v", name, err)
			}
			if msg := strings.ToLower(err.Error()); !strings.Contains(msg, "rights") && !strings.Contains(msg, "names") {
				t.Errorf("%s: error should name the field, got: %v", name, err)
			}
		})
	}

	_, err := Compare(dup{Rights: []right{{Name: "a", Scope: "x"}, {Name: "a", Scope: "y"}}}, dup{})
	if !errors.Is(err, ErrDuplicateIdentifier) || !strings.Contains(err.Error(), `"rights"`) {
		t.Errorf("expected ErrDuplicateIdentifier naming the path, got %v", err)
	}
}

// an array field is (part of) the identifier of its struct with the array_identifier option
func TestArrayIdentifier(t *testing.T) {
	type keyed struct {
		Key [2]string `cmp:"key,array_identifier"`
		Val int       `cmp:"val"`
	}
	diffs, err := Compare([]keyed{{[2]string{"a", "b"}, 1}}, []keyed{{[2]string{"a", "b"}, 2}})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"[a b]", "val"}, Left: 1, Right: 2}})

	// with a template and combined with another identifier field
	type combined struct {
		Key [2]string `cmp:"key,array_identifier:{{index .key 0}}-{{.id}}"`
		ID  int       `cmp:"id,identifier:{{index .key 0}}-{{.id}}"`
		Val int       `cmp:"val"`
	}
	diffs, err = Compare([]combined{{[2]string{"a", "b"}, 7, 1}}, []combined{{[2]string{"a", "b"}, 7, 2}})
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"a-7", "val"}, Left: 1, Right: 2}})

	// both options on one array field: the array identifies the struct, the template its elements
	type both struct {
		Rights [1]right `cmp:"rights,array_identifier:{{(index .rights 0).Name}},identifier:{{.Name}}"`
	}
	diffs, err = Compare(
		[]both{{Rights: [1]right{{Name: "r", Scope: "x"}}}},
		[]both{{Rights: [1]right{{Name: "r", Scope: "y"}}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"r", "rights", "r", "scope"}, Left: "x", Right: "y"}})

	// an array that cannot be a map key is an error instead of a panic
	type uncomparable struct {
		Rights [1]right `cmp:"rights,array_identifier"`
	}
	noPanic(t, func() {
		_, err := Compare([]uncomparable{{}}, []uncomparable{{}})
		if !errors.Is(err, ErrIdentifierTemplate) {
			t.Errorf("expected ErrIdentifierTemplate for an uncomparable identifier, got %v", err)
		}
	})

	// array_identifier is for arrays only
	type notArray struct {
		Key []string `cmp:"key,array_identifier"`
	}
	_, err = Compare([]notArray{{}}, []notArray{{}})
	if !errors.Is(err, ErrInvalidOption) {
		t.Errorf("expected ErrInvalidOption, got %v", err)
	}

	// a bare identifier on an array points to array_identifier
	type bare struct {
		Key [2]string `cmp:"key,identifier"`
	}
	_, err = Compare([]bare{{}}, []bare{{}})
	if !errors.Is(err, ErrIdentifierTemplate) || !strings.Contains(err.Error(), "array_identifier") {
		t.Errorf("expected ErrIdentifierTemplate mentioning array_identifier, got %v", err)
	}
}

// the option keeps its meaning on other fields: a templated identifier on scalar fields is still the
// identifier of the struct, and a struct with a container override does not become identifiable by it
func TestElemIdentifierDoesNotChangeFieldIdentifiers(t *testing.T) {
	type seat struct {
		Row    string  `cmp:"row,identifier:{{.row}}-{{.number}}"`
		Number int     `cmp:"number,identifier:{{.row}}-{{.number}}"`
		Rights []right `cmp:"rights,identifier:{{.Name}}"`
	}
	diffs, err := Compare(
		[]seat{{Row: "A", Number: 1, Rights: []right{{Name: "r", Scope: "x"}}}},
		[]seat{{Row: "A", Number: 1, Rights: []right{{Name: "r", Scope: "y"}}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"A-1", "rights", "r", "scope"}, Left: "x", Right: "y"}})

	// no field identifier: elements of the outer slice are matched by equality and index
	type user struct {
		Rights []right `cmp:"rights,identifier:{{.Name}}"`
	}
	diffs, err = Compare(
		[]user{{Rights: []right{{Name: "r", Scope: "x"}}}},
		[]user{{Rights: []right{{Name: "r", Scope: "y"}}}},
	)
	if err != nil {
		t.Fatal(err)
	}
	assertDiffs(t, diffs, Differences{{Type: CHANGED, Path: []string{"0", "rights", "r", "scope"}, Left: "x", Right: "y"}})
}
