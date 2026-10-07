package gompare

import "fmt"

// CompareOptsFunc configures a Comparer, see the With* functions
type CompareOptsFunc func(c *Comparer) error

// WithSliceOrdering determines whether the ordering of items in a slice results in a change
func WithSliceOrdering() CompareOptsFunc {
	return func(c *Comparer) error {
		c.config.sliceOrdering = true
		return nil
	}
}

// WithTagName sets the tag name to use when getting field names and options
func WithTagName(tag string) CompareOptsFunc {
	return func(c *Comparer) error {
		if tag == "" {
			return fmt.Errorf("%w: tag name must not be empty", ErrInvalidOption)
		}
		c.config.tagName = tag
		return nil
	}
}

// WithCombinedIdentifierJoinString allows to define custom identifier join string if templating is not used
func WithCombinedIdentifierJoinString(joinSep rune) CompareOptsFunc {
	return func(c *Comparer) error {
		c.config.combinedIdentifierJoinSep = joinSep
		return nil
	}
}

// WithSummarizeMissingStructs will add the whole struct as change and does not try to elems every struct field as change
func WithSummarizeMissingStructs() CompareOptsFunc {
	return func(c *Comparer) error {
		c.config.summarizeMissingStructs = true
		return nil
	}
}

// WithStructMapKeys will encode complex map keys with gob/base64 to be used as path element
func WithStructMapKeys() CompareOptsFunc {
	return func(c *Comparer) error {
		c.config.structMapKeys = true
		return nil
	}
}

// WithEmbeddedStructsAsField will put the embedded struct as path name
func WithEmbeddedStructsAsField() CompareOptsFunc {
	return func(c *Comparer) error {
		c.config.embeddedStructsAsFields = true
		return nil
	}
}

// WithAllowTypeMismatch reports two values of different kind (e.g. an int that became a string inside an
// interface or map[string]any) as CHANGED instead of aborting with ErrTypeMismatch
func WithAllowTypeMismatch() CompareOptsFunc {
	return func(c *Comparer) error {
		c.config.allowTypeMismatch = true
		return nil
	}
}

// WithSummarizeMissing reports a struct, slice, array or map that is missing on one side as one entry holding
// the whole value instead of one entry per field or element
func WithSummarizeMissing() CompareOptsFunc {
	return func(c *Comparer) error {
		c.config.summarizeMissing = true
		return nil
	}
}

// WithAllowDifferentStructs compares structs of different types field by field (matched by Go field name)
// instead of returning ErrTypeMismatch
func WithAllowDifferentStructs() CompareOptsFunc {
	return func(c *Comparer) error {
		c.config.allowDifferentStructs = true
		return nil
	}
}
