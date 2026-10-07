package gompare

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

type SimpleStructNoTag struct {
	Name  string
	Value int
}

type SimpleStructWithTag struct {
	Name  string `cmp:"name,identifier"`
	Value int    `cmp:"value"`
}

type NoIdentifierStruct struct {
	Value int `cmp:"value"`
}

type EmbeddedStruct struct {
	SimpleStructNoTag
	Baz bool `cmp:"baz"`
}

type EmbeddedStructWithID struct {
	SimpleStructNoTag
	Baz string `cmp:"baz,identifier"`
}

type ComplexStruct struct {
	ID              string                `cmp:"id"`
	Name            string                `cmp:"name"`
	Value           int                   `cmp:"value"`
	Bool            bool                  `cmp:"bool"`
	Values          []string              `cmp:"values"`
	Map             map[string]string     `cmp:"map"`
	Time            time.Time             `cmp:"time"`
	Pointer         *string               `cmp:"pointer"`
	Ignored         bool                  `cmp:"-"`
	Identifiables   []SimpleStructWithTag `cmp:"identifiables"`
	Unidentifiables []NoIdentifierStruct  `cmp:"unidentifiables"`
	private         int                   `cmp:"private"`
}

type RealWorldSubStruct struct {
	Name string `cmp:"name"`
	ID   int64  `cmp:"-,identifier"`
}

type RealWorldSubStructCombinedID struct {
	Name string `cmp:"name"`
	ID   int64  `cmp:"-,identifier:{{.ID}}/{{.SID}},R"`
	SID  int64  `cmp:"-,identifier:{{.ID}}/{{.SID}},T"`
}

type RealWorldSubStructCombinedIDNoTemplate struct {
	Name string `cmp:"name"`
	ID   int64  `cmp:"-,identifier"`
	SID  int64  `cmp:"-,identifier"`
}

type RealWorldStructCombinedID struct {
	Name   string                          `cmp:"name,identifier"`
	Value  int                             `cmp:"value"`
	Addons []*RealWorldSubStructCombinedID `cmp:"addons"`
}

type RealWorldStruct struct {
	Name   string                `cmp:"name,identifier"`
	Value  int                   `cmp:"value"`
	Addons []*RealWorldSubStruct `cmp:"addons"`
}

type CustomTagged struct {
	Name  string `diff:"name,identifier"`
	Value int    `diff:"value"`
	Skip  int    `diff:"-"`
}

var testTimeA, _ = time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
var testTimeB, _ = time.Parse(time.RFC3339, "2007-01-02T15:04:05Z")

func getStringPointer(s string) *string {
	return &s
}

func TestCompare(t *testing.T) {
	cases := []struct {
		Name        string
		LEFT, RIGHT any
		Changes     Differences
		Error       error
		Options     []CompareOptsFunc
	}{
		{
			"uint-equal",
			uint(1),
			uint(1),
			Differences{},
			nil,
			nil,
		},
		{
			"uint-not-equal",
			uint(1),
			uint(2),
			Differences{
				Difference{Type: CHANGED, Path: []string{}, Left: uint(1), Right: uint(2)},
			},
			nil,
			nil,
		},
		{
			"int-equal",
			int(1),
			int(1),
			Differences{},
			nil,
			nil,
		},
		{
			"int-not-equal",
			int(1),
			int(2),
			Differences{
				Difference{Type: CHANGED, Path: []string{}, Left: int(1), Right: int(2)},
			},
			nil,
			nil,
		},
		{
			"float-equal",
			float64(1.1),
			float64(1.1),
			Differences{},
			nil,
			nil,
		},
		{
			"float-not-equal",
			float64(1.1),
			float64(2.2),
			Differences{
				Difference{Type: CHANGED, Path: []string{}, Left: float64(1.1), Right: float64(2.2)},
			},
			nil,
			nil,
		},
		{
			"string-equal",
			"hello",
			"hello",
			Differences{},
			nil,
			nil,
		},
		{
			"string-not-equal",
			"hello",
			"world",
			Differences{
				Difference{Type: CHANGED, Path: []string{}, Left: "hello", Right: "world"},
			},
			nil,
			nil,
		},
		{
			"time-equal",
			testTimeA,
			testTimeA,
			Differences{},
			nil,
			nil,
		},
		{
			"time-not-equal", testTimeA, testTimeB,
			Differences{
				Difference{Type: CHANGED, Path: []string{}, Left: testTimeA, Right: testTimeB},
			},
			nil,
			nil,
		},
		{
			"SimpleStructNoTag-equal",
			SimpleStructNoTag{Name: "test LEFT", Value: 123},
			SimpleStructNoTag{Name: "test LEFT", Value: 123},
			Differences{},
			nil,
			nil,
		},
		{
			"SimpleStructNoTag-not-equal-name",
			SimpleStructNoTag{Name: "test LEFT", Value: 123},
			SimpleStructNoTag{Name: "test RIGHT", Value: 123},
			Differences{
				Difference{Type: CHANGED, Path: []string{"Name"}, Left: "test LEFT", Right: "test RIGHT"},
			},
			nil,
			nil,
		},
		{
			"SimpleStructNoTag-not-equal-name-and-value",
			SimpleStructNoTag{Name: "test LEFT", Value: 123},
			SimpleStructNoTag{Name: "test RIGHT", Value: 456},
			Differences{
				Difference{Type: CHANGED, Path: []string{"Name"}, Left: "test LEFT", Right: "test RIGHT"},
				Difference{Type: CHANGED, Path: []string{"Value"}, Left: 123, Right: 456},
			},
			nil,
			nil,
		},
		{
			"SimpleStructWithTag-equal",
			SimpleStructWithTag{Name: "test LEFT", Value: 123},
			SimpleStructWithTag{Name: "test LEFT", Value: 123},
			Differences{},
			nil,
			nil,
		},
		{
			"SimpleStructWithTag-not-equal-name-only",
			SimpleStructWithTag{Name: "test LEFT", Value: 123},
			SimpleStructWithTag{Name: "test RIGHT", Value: 123},
			Differences{
				Difference{Type: CHANGED, Path: []string{"name"}, Left: "test LEFT", Right: "test RIGHT"},
			},
			nil,
			nil,
		},
		{
			"SimpleStructWithTag-not-equal",
			SimpleStructWithTag{Name: "test LEFT", Value: 123},
			SimpleStructWithTag{Name: "test RIGHT", Value: 456},
			Differences{
				Difference{Type: CHANGED, Path: []string{"name"}, Left: "test LEFT", Right: "test RIGHT"},
				Difference{Type: CHANGED, Path: []string{"value"}, Left: 123, Right: 456},
			},
			nil,
			nil,
		},
		{
			"different-structs-tagged-vs-untagged",
			SimpleStructWithTag{Name: "test LEFT", Value: 123},
			SimpleStructNoTag{Name: "test LEFT", Value: 123},
			Differences{},
			ErrTypeMismatch,
			nil,
		},
		{
			"different-structs-untagged-vs-tagged",
			SimpleStructNoTag{Name: "test LEFT", Value: 123},
			SimpleStructWithTag{Name: "test LEFT", Value: 123},
			Differences{},
			ErrTypeMismatch,
			nil,
		},
		{
			"different-structs-allowed",
			SimpleStructNoTag{Name: "test LEFT", Value: 123},
			SimpleStructWithTag{Name: "test RIGHT", Value: 123},
			Differences{
				Difference{Type: CHANGED, Path: []string{"Name"}, Left: "test LEFT", Right: "test RIGHT"},
			},
			nil,
			[]CompareOptsFunc{WithAllowDifferentStructs()},
		},
		{
			"int-slice-insert",
			[]int{1, 2, 3},
			[]int{1, 2, 3, 4},
			Differences{
				Difference{Type: ADDED, Path: []string{"3"}, Right: 4},
			},
			nil,
			nil,
		},
		{
			"int-array-insert",
			[3]int{1, 2, 3},
			[4]int{1, 2, 3, 4},
			Differences{
				Difference{Type: ADDED, Path: []string{"3"}, Right: 4},
			},
			nil,
			nil,
		},
		{
			"int-slice-delete",
			[]int{1, 2, 3},
			[]int{1, 3},
			Differences{
				Difference{Type: REMOVED, Path: []string{"1"}, Left: 2},
			},
			nil,
			nil,
		},
		{
			"int-array-delete",
			[3]int{1, 2, 3},
			[2]int{1, 3},
			Differences{
				Difference{Type: REMOVED, Path: []string{"1"}, Left: 2},
			},
			nil,
			nil,
		},
		{
			"uint-slice-insert",
			[]uint{1, 2, 3},
			[]uint{1, 2, 3, 4},
			Differences{
				Difference{Type: ADDED, Path: []string{"3"}, Right: uint(4)},
			},
			nil,
			nil,
		},
		{
			"uint-array-insert",
			[3]uint{1, 2, 3},
			[4]uint{1, 2, 3, 4},
			Differences{
				Difference{Type: ADDED, Path: []string{"3"}, Right: uint(4)},
			},
			nil,
			nil,
		},
		{
			"uint-slice-delete",
			[]uint{1, 2, 3},
			[]uint{1, 3},
			Differences{
				Difference{Type: REMOVED, Path: []string{"1"}, Left: uint(2)},
			},
			nil,
			nil,
		},
		{
			"uint-array-delete",
			[3]uint{1, 2, 3},
			[2]uint{1, 3},
			Differences{
				Difference{Type: REMOVED, Path: []string{"1"}, Left: uint(2)},
			},
			nil,
			nil,
		},
		{
			"string-slice-insert",
			[]string{"1", "2", "3"},
			[]string{"1", "2", "3", "4"},
			Differences{
				Difference{Type: ADDED, Path: []string{"3"}, Right: "4"},
			},
			nil,
			nil,
		},
		{
			"string-array-insert",
			[3]string{"1", "2", "3"},
			[4]string{"1", "2", "3", "4"},
			Differences{
				Difference{Type: ADDED, Path: []string{"3"}, Right: "4"},
			},
			nil,
			nil,
		},
		{
			"string-slice-delete",
			[]string{"1", "2", "3"},
			[]string{"1", "3"},
			Differences{
				Difference{Type: REMOVED, Path: []string{"1"}, Left: "2"},
			},
			nil,
			nil,
		},
		{
			"string-array-delete",
			[3]string{"1", "2", "3"},
			[2]string{"1", "3"},
			Differences{
				Difference{Type: REMOVED, Path: []string{"1"}, Left: "2"},
			},
			nil,
			nil,
		},
		{
			"string-slice-insert-delete",
			[]string{"1", "2", "3"},
			[]string{"1", "3", "4"},
			Differences{
				Difference{Type: REMOVED, Path: []string{"1"}, Left: "2"},
				Difference{Type: ADDED, Path: []string{"2"}, Right: "4"},
			},
			nil,
			nil,
		},
		{
			"string-array-insert-delete",
			[3]string{"1", "2", "3"},
			[3]string{"1", "3", "4"},
			Differences{
				Difference{Type: REMOVED, Path: []string{"1"}, Left: "2"},
				Difference{Type: ADDED, Path: []string{"2"}, Right: "4"},
			},
			nil,
			nil,
		},
		{
			"isComparable-slice-insert",
			[]SimpleStructWithTag{{"one", 1}},
			[]SimpleStructWithTag{{"one", 1}, {"two", 2}},
			Differences{
				Difference{Type: ADDED, Path: []string{"two", "name"}, Right: "two"},
				Difference{Type: ADDED, Path: []string{"two", "value"}, Right: 2},
			},
			nil,
			nil,
		},
		{
			"isComparable-array-insert",
			[1]SimpleStructWithTag{{"one", 1}},
			[2]SimpleStructWithTag{{"one", 1}, {"two", 2}},
			Differences{
				Difference{Type: ADDED, Path: []string{"two", "name"}, Right: "two"},
				Difference{Type: ADDED, Path: []string{"two", "value"}, Right: 2},
			},
			nil,
			nil,
		},
		{
			"isComparable-slice-delete",
			[]SimpleStructWithTag{{"one", 1}, {"two", 2}},
			[]SimpleStructWithTag{{"one", 1}},
			Differences{
				Difference{Type: REMOVED, Path: []string{"two", "name"}, Left: "two"},
				Difference{Type: REMOVED, Path: []string{"two", "value"}, Left: 2},
			},
			nil,
			nil,
		},
		{
			"isComparable-array-delete",
			[2]SimpleStructWithTag{{"one", 1}, {"two", 2}},
			[1]SimpleStructWithTag{{"one", 1}},
			Differences{
				Difference{Type: REMOVED, Path: []string{"two", "name"}, Left: "two"},
				Difference{Type: REMOVED, Path: []string{"two", "value"}, Left: 2},
			},
			nil,
			nil,
		},
		{
			"isComparable-array-delete-embedded",
			[2]EmbeddedStructWithID{{SimpleStructNoTag{Name: "f1", Value: 1}, "one"}, {SimpleStructNoTag{Name: "f2", Value: 2}, "two"}},
			[1]EmbeddedStructWithID{{SimpleStructNoTag{Name: "f1", Value: 1}, "one"}},
			Differences{
				Difference{Type: REMOVED, Path: []string{"two", "SimpleStructNoTag", "Name"}, Left: "f2"},
				Difference{Type: REMOVED, Path: []string{"two", "SimpleStructNoTag", "Value"}, Left: 2},
				Difference{Type: REMOVED, Path: []string{"two", "baz"}, Left: "two"},
			},
			nil,
			[]CompareOptsFunc{WithEmbeddedStructsAsField()},
		},
		{
			"isComparable-array-delete-embedded-wo-opt",
			[2]EmbeddedStructWithID{{SimpleStructNoTag{Name: "f1", Value: 1}, "one"}, {SimpleStructNoTag{Name: "f2", Value: 2}, "two"}},
			[1]EmbeddedStructWithID{{SimpleStructNoTag{Name: "f1", Value: 1}, "one"}},
			Differences{
				Difference{Type: REMOVED, Path: []string{"two", "Name"}, Left: "f2"},
				Difference{Type: REMOVED, Path: []string{"two", "Value"}, Left: 2},
				Difference{Type: REMOVED, Path: []string{"two", "baz"}, Left: "two"},
			},
			nil,
			nil,
		},
		{
			"isComparable-slice-update",
			[]SimpleStructWithTag{{"one", 1}},
			[]SimpleStructWithTag{{"one", 50}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"one", "value"}, Left: 1, Right: 50},
			},
			nil,
			nil,
		},
		{
			"isComparable-array-update",
			[1]SimpleStructWithTag{{"one", 1}},
			[1]SimpleStructWithTag{{"one", 50}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"one", "value"}, Left: 1, Right: 50},
			},
			nil,
			nil,
		},
		{
			"map-slice-insert",
			[]map[string]string{{"test": "123"}},
			[]map[string]string{{"test": "123", "tset": "456"}},
			Differences{
				Difference{Type: ADDED, Path: []string{"0", "tset"}, Right: "456"},
			},
			nil,
			nil,
		},
		{
			"map-array-insert",
			[1]map[string]string{{"test": "123"}},
			[1]map[string]string{{"test": "123", "tset": "456"}},
			Differences{
				Difference{Type: ADDED, Path: []string{"0", "tset"}, Right: "456"},
			},
			nil,
			nil,
		},
		{
			"map-slice-update",
			[]map[string]string{{"test": "123"}},
			[]map[string]string{{"test": "456"}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"0", "test"}, Left: "123", Right: "456"},
			},
			nil,
			nil,
		},
		{
			"map-array-update",
			[1]map[string]string{{"test": "123"}},
			[1]map[string]string{{"test": "456"}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"0", "test"}, Left: "123", Right: "456"},
			},
			nil,
			nil,
		},
		{
			"map-slice-delete",
			[]map[string]string{{"test": "123", "tset": "456"}},
			[]map[string]string{{"test": "123"}},
			Differences{
				Difference{Type: REMOVED, Path: []string{"0", "tset"}, Left: "456"},
			},
			nil,
			nil,
		},
		{
			"map-array-delete",
			[1]map[string]string{{"test": "123", "tset": "456"}},
			[1]map[string]string{{"test": "123"}},
			Differences{
				Difference{Type: REMOVED, Path: []string{"0", "tset"}, Left: "456"},
			},
			nil,
			nil,
		},
		{
			"map-interface-slice-update",
			[]map[string]interface{}{{"test": nil}},
			[]map[string]interface{}{{"test": "456"}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"0", "test"}, Left: nil, Right: "456"},
			},
			nil,
			nil,
		},
		{
			"map-interface-array-update",
			[1]map[string]interface{}{{"test": nil}},
			[1]map[string]interface{}{{"test": "456"}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"0", "test"}, Left: nil, Right: "456"},
			},
			nil,
			nil,
		},
		{
			"map-nil",
			map[string]string{"one": "test"},
			nil,
			Differences{
				Difference{Type: REMOVED, Path: []string{"one"}, Left: "test", Right: nil},
			},
			nil,
			nil,
		},
		{
			"nil-map",
			nil,
			map[string]string{"one": "test"},
			Differences{
				Difference{Type: ADDED, Path: []string{"one"}, Left: nil, Right: "test"},
			},
			nil,
			nil,
		},
		{
			"nested-map-insert",
			map[string]map[string]string{"a": {"test": "123"}},
			map[string]map[string]string{"a": {"test": "123", "tset": "456"}},
			Differences{
				Difference{Type: ADDED, Path: []string{"a", "tset"}, Right: "456"},
			},
			nil,
			nil,
		},
		{
			"nested-map-interface-insert",
			map[string]map[string]interface{}{"a": {"test": "123"}},
			map[string]map[string]interface{}{"a": {"test": "123", "tset": "456"}},
			Differences{
				Difference{Type: ADDED, Path: []string{"a", "tset"}, Right: "456"},
			},
			nil,
			nil,
		},
		{
			"nested-map-update",
			map[string]map[string]string{"a": {"test": "123"}},
			map[string]map[string]string{"a": {"test": "456"}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"a", "test"}, Left: "123", Right: "456"},
			},
			nil,
			nil,
		},
		{
			"nested-map-delete",
			map[string]map[string]string{"a": {"test": "123"}},
			map[string]map[string]string{"a": {}},
			Differences{
				Difference{Type: REMOVED, Path: []string{"a", "test"}, Left: "123", Right: nil},
			},
			nil,
			nil,
		},
		{
			"nested-slice-insert",
			map[string][]int{"a": {1, 2, 3}},
			map[string][]int{"a": {1, 2, 3, 4}},
			Differences{
				Difference{Type: ADDED, Path: []string{"a", "3"}, Right: 4},
			},
			nil,
			nil,
		},
		{
			"nested-array-insert",
			map[string][3]int{"a": {1, 2, 3}},
			map[string][4]int{"a": {1, 2, 3, 4}},
			Differences{
				Difference{Type: ADDED, Path: []string{"a", "3"}, Right: 4},
			},
			nil,
			nil,
		},
		{
			"nested-slice-update",
			map[string][]int{"a": {1, 2, 3}},
			map[string][]int{"a": {1, 4, 3}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"a", "1"}, Left: 2, Right: 4},
			},
			nil,
			nil,
		},
		{
			"nested-array-update",
			map[string][3]int{"a": {1, 2, 3}},
			map[string][3]int{"a": {1, 4, 3}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"a", "1"}, Left: 2, Right: 4},
			},
			nil,
			nil,
		},
		{
			"nested-slice-delete",
			map[string][]int{"a": {1, 2, 3}},
			map[string][]int{"a": {1, 3}},
			Differences{
				Difference{Type: REMOVED, Path: []string{"a", "1"}, Left: 2, Right: nil},
			},
			nil,
			nil,
		},
		{
			"nested-array-delete",
			map[string][3]int{"a": {1, 2, 3}},
			map[string][2]int{"a": {1, 3}},
			Differences{
				Difference{Type: REMOVED, Path: []string{"a", "1"}, Left: 2, Right: nil},
			},
			nil,
			nil,
		},

		{
			"struct-string-update",
			ComplexStruct{Name: "one"},
			ComplexStruct{Name: "two"},
			Differences{
				Difference{Type: CHANGED, Path: []string{"name"}, Left: "one", Right: "two"},
			},
			nil,
			nil,
		},
		{
			"struct-int-update",
			ComplexStruct{Value: 1},
			ComplexStruct{Value: 50},
			Differences{
				Difference{Type: CHANGED, Path: []string{"value"}, Left: 1, Right: 50},
			},
			nil,
			nil,
		},
		{
			"struct-bool-update",
			ComplexStruct{Bool: true},
			ComplexStruct{Bool: false},
			Differences{
				Difference{Type: CHANGED, Path: []string{"bool"}, Left: true, Right: false},
			},
			nil,
			nil,
		},
		{
			"struct-time-update",
			ComplexStruct{},
			ComplexStruct{Time: testTimeA},
			Differences{
				Difference{Type: CHANGED, Path: []string{"time"}, Left: time.Time{}, Right: testTimeA},
			},
			nil,
			nil,
		},
		{
			"struct-map-update",
			ComplexStruct{Map: map[string]string{"test": "123"}},
			ComplexStruct{Map: map[string]string{"test": "456"}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"map", "test"}, Left: "123", Right: "456"},
			},
			nil,
			nil,
		},
		{
			"struct-string-pointer-update",
			ComplexStruct{Pointer: getStringPointer("test")},
			ComplexStruct{Pointer: getStringPointer("test2")},
			Differences{
				Difference{Type: CHANGED, Path: []string{"pointer"}, Left: "test", Right: "test2"},
			},
			nil,
			nil,
		},
		{
			"struct-nil-string-pointer-update",
			ComplexStruct{Pointer: nil},
			ComplexStruct{Pointer: getStringPointer("test")},
			Differences{
				Difference{Type: CHANGED, Path: []string{"pointer"}, Left: nil, Right: "test"},
			},
			nil,
			nil,
		},
		{
			"struct-generic-slice-insert",
			ComplexStruct{Values: []string{"one"}},
			ComplexStruct{Values: []string{"one", "two"}},
			Differences{
				Difference{Type: ADDED, Path: []string{"values", "1"}, Left: nil, Right: "two"},
			},
			nil,
			nil,
		},
		{
			"struct-generic-slice-delete",
			ComplexStruct{Values: []string{"one", "two"}},
			ComplexStruct{Values: []string{"one"}},
			Differences{
				Difference{Type: REMOVED, Path: []string{"values", "1"}, Left: "two", Right: nil},
			},
			nil,
			nil,
		},
		{
			"omittable",
			ComplexStruct{Ignored: false},
			ComplexStruct{Ignored: true},
			Differences{},
			nil,
			nil,
		},
		{
			"slice-duplicate-items",
			[]int{1},
			[]int{1, 1},
			Differences{
				Difference{Type: ADDED, Path: []string{"1"}, Left: nil, Right: 1},
			},
			nil,
			nil,
		},
		{
			"mixed-slice-map",
			[]map[string]interface{}{{"name": "name1", "type": []string{"null", "string"}}},
			[]map[string]interface{}{{"name": "name1", "type": []string{"null", "int"}}, {"name": "name2", "type": []string{"null", "string"}}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"0", "type", "1"}, Left: "string", Right: "int"},
				Difference{Type: ADDED, Path: []string{"1", "name"}, Left: nil, Right: "name2"},
				Difference{Type: ADDED, Path: []string{"1", "type", "0"}, Left: nil, Right: "null"},
				Difference{Type: ADDED, Path: []string{"1", "type", "1"}, Left: nil, Right: "string"},
			},
			nil,
			nil,
		},
		{
			"private-struct-field",
			ComplexStruct{private: 1},
			ComplexStruct{private: 4},
			Differences{
				Difference{Type: CHANGED, Path: []string{"private"}, Left: 1, Right: 4},
			},
			nil,
			nil,
		},
		{
			"embedded-struct-field",
			EmbeddedStruct{SimpleStructNoTag{Name: "a", Value: 2}, true},
			EmbeddedStruct{SimpleStructNoTag{Name: "b", Value: 3}, false},
			Differences{
				Difference{Type: CHANGED, Path: []string{"Name"}, Left: "a", Right: "b"},
				Difference{Type: CHANGED, Path: []string{"Value"}, Left: 2, Right: 3},
				Difference{Type: CHANGED, Path: []string{"baz"}, Left: true, Right: false},
			},
			nil,
			nil,
		},
		{
			"embedded-struct-field-as-extra-field",
			EmbeddedStruct{SimpleStructNoTag{Name: "a", Value: 2}, true},
			EmbeddedStruct{SimpleStructNoTag{Name: "b", Value: 3}, false},
			Differences{
				Difference{Type: CHANGED, Path: []string{"SimpleStructNoTag", "Name"}, Left: "a", Right: "b"},
				Difference{Type: CHANGED, Path: []string{"SimpleStructNoTag", "Value"}, Left: 2, Right: 3},
				Difference{Type: CHANGED, Path: []string{"baz"}, Left: true, Right: false},
			},
			nil,
			[]CompareOptsFunc{WithEmbeddedStructsAsField()},
		},
		{
			"real-world-struct",
			RealWorldStruct{Name: "TestA", Value: 1, Addons: []*RealWorldSubStruct{{Name: "Sub1", ID: 10}, {Name: "Sub2", ID: 20}}},
			RealWorldStruct{Name: "TestB", Value: 1, Addons: []*RealWorldSubStruct{{Name: "Sub1", ID: 10}, {Name: "Sub3", ID: 30}}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"name"}, Left: "TestA", Right: "TestB"},
				Difference{Type: REMOVED, Path: []string{"addons", "20", "name"}, Left: "Sub2"},
				Difference{Type: ADDED, Path: []string{"addons", "30", "name"}, Right: "Sub3"},
			},
			nil,
			nil,
		},
		{
			"real-world-struct-from-nil",
			nil,
			RealWorldStruct{Name: "TestB", Value: 1, Addons: []*RealWorldSubStruct{{Name: "Sub1", ID: 10}, {Name: "Sub3", ID: 30}}},
			Differences{
				Difference{Type: ADDED, Path: []string{"name"}, Right: "TestB"},
				Difference{Type: ADDED, Path: []string{"value"}, Right: 1},
				Difference{Type: ADDED, Path: []string{"addons", "10", "name"}, Right: "Sub1"},
				Difference{Type: ADDED, Path: []string{"addons", "30", "name"}, Right: "Sub3"},
			},
			nil,
			nil,
		},
		{
			"real-world-struct-from-nil-combined-identifier",
			nil,
			RealWorldStructCombinedID{Name: "TestB", Value: 1, Addons: []*RealWorldSubStructCombinedID{{Name: "Sub1", SID: 11, ID: 10}, {Name: "Sub3", SID: 33, ID: 30}}},
			Differences{
				Difference{Type: ADDED, Path: []string{"name"}, Right: "TestB"},
				Difference{Type: ADDED, Path: []string{"value"}, Right: 1},
				Difference{Type: ADDED, Path: []string{"addons", "10/11", "name"}, Right: "Sub1"},
				Difference{Type: ADDED, Path: []string{"addons", "30/33", "name"}, Right: "Sub3"},
			},
			nil,
			nil,
		},
		{
			"custom-tags",
			[]CustomTagged{{Name: "a", Value: 1, Skip: 1}},
			[]CustomTagged{{Name: "a", Value: 2, Skip: 2}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"a", "value"}, Left: 1, Right: 2},
			},
			nil,
			[]CompareOptsFunc{WithTagName("diff")},
		},
		{
			"slice-ordering",
			[]string{"a", "b"},
			[]string{"b", "a"},
			Differences{
				Difference{Type: CHANGED, Path: []string{"0"}, Left: "a", Right: "b"},
				Difference{Type: CHANGED, Path: []string{"1"}, Left: "b", Right: "a"},
			},
			nil,
			[]CompareOptsFunc{WithSliceOrdering()},
		},
		{
			"summarize-missing-structs",
			nil,
			SimpleStructNoTag{Name: "a", Value: 1},
			Differences{
				Difference{Type: ADDED, Path: []string{}, Right: SimpleStructNoTag{Name: "a", Value: 1}},
			},
			nil,
			[]CompareOptsFunc{WithSummarizeMissingStructs()},
		},
		{
			"combined-identifier-join-string",
			[]RealWorldSubStructCombinedIDNoTemplate{{ID: 1, SID: 2, Name: "a"}},
			[]RealWorldSubStructCombinedIDNoTemplate{{ID: 1, SID: 2, Name: "b"}},
			Differences{
				Difference{Type: CHANGED, Path: []string{"1#2", "name"}, Left: "a", Right: "b"},
			},
			nil,
			[]CompareOptsFunc{WithCombinedIdentifierJoinString('#')},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			cl, err := Compare(tc.LEFT, tc.RIGHT, tc.Options...)

			if !errors.Is(err, tc.Error) {
				t.Errorf("unexpected error - got: %v, wanted: %v", err, tc.Error)
			}
			if len(cl) != len(tc.Changes) {
				t.Fatalf("unexpected number of differences - got: %d, wanted: %d\n got:  %+v\n want: %+v", len(cl), len(tc.Changes), cl, tc.Changes)
			}

			for i, c := range cl {
				want := tc.Changes[i]
				if want.Type != c.Type {
					t.Errorf("unexpected type - got: %s, wanted: %s", c.Type, want.Type)
				}
				if strings.Join(want.Path, "/") != strings.Join(c.Path, "/") {
					t.Errorf("unexpected path - got: %v, wanted: %v", c.Path, want.Path)
				}
				if !reflect.DeepEqual(want.Left, c.Left) {
					t.Errorf("unexpected LEFT - got: %v, wanted: %v", c.Left, want.Left)
				}
				if !reflect.DeepEqual(want.Right, c.Right) {
					t.Errorf("unexpected RIGHT - got: %v, wanted: %v", c.Right, want.Right)
				}
			}
		})
	}
}
