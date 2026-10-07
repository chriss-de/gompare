package gompare

import (
	"reflect"
)

// comparableListEntry is an object holding two items that can be compared
type comparableListEntry struct {
	LEFT, RIGHT *reflect.Value
}

// comparableList stores an indexed elems of comparableListEntry items
type comparableList struct {
	elems map[any]*comparableListEntry
	keys  []any
}

// newComparableList returns a new comparableList
func newComparableList() *comparableList {
	return &comparableList{
		elems: make(map[any]*comparableListEntry),
		keys:  make([]any, 0),
	}
}

// indexKey is the key of a slice element that has no identifier
type indexKey int

// addLeft adds a value with key to the left side for comparison.
// It returns false if the left side already holds a value for key.
func (cl *comparableList) addLeft(key any, val *reflect.Value) bool {
	if cl.elems[key] == nil {
		cl.elems[key] = &comparableListEntry{}
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
func (cl *comparableList) addRight(key any, val *reflect.Value) bool {
	if cl.elems[key] == nil {
		cl.elems[key] = &comparableListEntry{}
		cl.keys = append(cl.keys, key)
	}
	if cl.elems[key].RIGHT != nil {
		return false
	}
	cl.elems[key].RIGHT = val
	return true
}
