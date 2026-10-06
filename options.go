package gompare

type CompareOptsFunc func(d *Comparer) error

// WithSliceOrdering determines whether the ordering of items in a slice results in a change
func WithSliceOrdering() func(c *Comparer) error {
	return func(c *Comparer) error {
		c.config.sliceOrdering = true
		return nil
	}
}

// WithTagName sets the tag name to use when getting field names and options
func WithTagName(tag string) func(c *Comparer) error {
	return func(c *Comparer) error {
		c.config.tagName = tag
		return nil
	}
}

// WithCombinedIdentifierJoinString allows to define custom identifier join string if templating is not used
func WithCombinedIdentifierJoinString(joinSep rune) func(c *Comparer) error {
	return func(c *Comparer) error {
		c.config.combinedIdentifierJoinSep = joinSep
		return nil
	}
}

// WithSummarizeMissingStructs will add the whole struct as change and does not try to elems every struct field as change
func WithSummarizeMissingStructs() func(c *Comparer) error {
	return func(c *Comparer) error {
		c.config.summarizeMissingStructs = true
		return nil
	}
}

// WithStructMapKeys will encode complex map keys with gob/base64 to be used as path element
func WithStructMapKeys() func(c *Comparer) error {
	return func(c *Comparer) error {
		c.config.structMapKeys = true
		return nil
	}
}

// WithEmbeddedStructsAsField will put the embedded struct as path name
func WithEmbeddedStructsAsField() func(c *Comparer) error {
	return func(c *Comparer) error {
		c.config.embeddedStructsAsFields = true
		return nil
	}
}

// WithAllowTypeMismatch reports two values of different kind (e.g. an int that became a string inside an
// interface or map[string]any) as CHANGED instead of aborting with ErrTypeMismatch
func WithAllowTypeMismatch() func(c *Comparer) error {
	return func(c *Comparer) error {
		c.config.allowTypeMismatch = true
		return nil
	}
}
