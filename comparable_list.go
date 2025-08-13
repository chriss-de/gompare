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

// addLeft adds a value with key to the left side for comparison
func (cl *ComparableList) addLeft(key any, val *reflect.Value) {
	if (*cl).elems[key] == nil {
		(*cl).elems[key] = &ComparableListEntry{}
		(*cl).keys = append((*cl).keys, key)
	}
	(*cl).elems[key].LEFT = val
}

// addRight adds a value with key to the right side for comparison
func (cl *ComparableList) addRight(key any, val *reflect.Value) {
	if (*cl).elems[key] == nil {
		(*cl).elems[key] = &ComparableListEntry{}
		(*cl).keys = append((*cl).keys, key)
	}

	(*cl).elems[key].RIGHT = val
}
