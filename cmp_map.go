package gompare

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"reflect"
)

// cmpMap compare's two maps
// If one is empty/invalid we add all keys as ADDED/REMOVED
// otherwise compare each key
func (c *Comparer) cmpMap(path []string, left, right reflect.Value) error {
	if left.Kind() == reflect.Invalid {
		return c.cmpMapValuesForInvalid(ADDED, path, right)
	}
	if right.Kind() == reflect.Invalid {
		return c.cmpMapValuesForInvalid(REMOVED, path, left)
	}

	cmpList := NewComparableList()

	for _, k := range sortedMapKeys(left) {
		leftElem := left.MapIndex(k)
		cmpList.addLeft(getAsAny(k), &leftElem)
	}

	for _, k := range sortedMapKeys(right) {
		rightElem := right.MapIndex(k)
		cmpList.addRight(getAsAny(k), &rightElem)
	}

	return c.processComparableList(path, cmpList)
}

// cmpMapValuesForInvalid is used by cmpMap for maps that are empty/invalid to note all Differences
func (c *Comparer) cmpMapValuesForInvalid(dt DiffType, path []string, val reflect.Value) error {
	if dt != ADDED && dt != REMOVED {
		return ErrInvalidChangeType
	}

	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}

	if val.Kind() != reflect.Map {
		return ErrTypeMismatch
	}

	// compare on a clone so only the Differences of this map get patched below
	var nc *Comparer = c.clone()
	missing := reflect.Value{}

	for _, k := range sortedMapKeys(val) {
		ae := val.MapIndex(k)

		var err error

		// if configured we encode the map key with gob/base64
		// otherwise just print it as string
		// strings won't work on complex keys that cannot be stringified
		if c.config.structMapKeys {
			var bWriter = new(bytes.Buffer)
			if err = gob.NewEncoder(bWriter).Encode(getAsAny(k)); err == nil {
				key := base64.RawStdEncoding.EncodeToString(bWriter.Bytes())
				err = nc.compare(copyAppend(path, key), missing, ae)
			}
		} else {
			err = nc.compare(copyAppend(path, fmt.Sprint(getAsAny(k))), missing, ae)
		}
		if err != nil {
			return err
		}
	}

	// we have to patch Differences to get the right Difference result
	for i := 0; i < len(nc.differences); i++ {
		diff := nc.differences[i]
		diff.patchLeftAndRight(dt)
		c.differences = append(c.differences, diff)
	}

	return nil
}
