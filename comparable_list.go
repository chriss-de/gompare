package gompare

import (
	"reflect"
)

// ComparableListEntry is an object holding two items that can be compared
type ComparableListEntry struct {
	LEFT, RIGHT *reflect.Value
}

// ComparableList stores an indexed elems of ComparableListEntry items
type ComparableList struct {
	elems map[any]*ComparableListEntry
	keys  []any
}

// NewComparableList returns a new ComparableList
func NewComparableList() *ComparableList {
	return &ComparableList{
		elems: make(map[any]*ComparableListEntry),
		keys:  make([]any, 0),
	}
}

// indexKey is the key of a slice element that has no identifier
type indexKey int

// addLeft adds a value with key to the left side for comparison.
// It returns false if the left side already holds a value for key.
func (cl *ComparableList) addLeft(key any, val *reflect.Value) bool {
	if cl.elems[key] == nil {
		cl.elems[key] = &ComparableListEntry{}
		cl.keys = append(cl.keys, key)
	}
	if cl.elems[key].LEFT != nil {
		return false
	}
	cl.elems[key].LEFT = val
	return true
}

// addRight adds a value with key to the right side for comparison.
// It returns false if the right side already holds a value for key.
func (cl *ComparableList) addRight(key any, val *reflect.Value) bool {
	if cl.elems[key] == nil {
		cl.elems[key] = &ComparableListEntry{}
		cl.keys = append(cl.keys, key)
	}
	if cl.elems[key].RIGHT != nil {
		return false
	}
	cl.elems[key].RIGHT = val
	return true
}
