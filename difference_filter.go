package gompare

import "slices"

// WhereOr combines multiple DifferenceFilterFunc's as GetDifferences accepts DifferenceFilterFunc's but the result is AND'ed
// With WhereOr you can combine almost any logic construct to filter Differences
func WhereOr(filterFunc ...DifferenceFilterFunc) DifferenceFilterFunc {
	return func(d Difference) bool {
		for _, ff := range filterFunc {
			if ff(d) {
				return true
			}
		}
		return false
	}
}

// WherePath filters Differences - if p is in Difference path
func WherePath(p string) DifferenceFilterFunc {
	return func(d Difference) bool {
		return slices.Contains(d.Path, p)
	}
}

// WherePathAt filters Differences - if p is at index in Difference path
func WherePathAt(p string, idx int) DifferenceFilterFunc {
	return func(d Difference) bool {
		return len(d.Path) > idx && d.Path[idx] == p
	}
}

// WherePathDepth filters Differences - if Difference path is of length l
func WherePathDepth(l int) DifferenceFilterFunc {
	return func(d Difference) bool {
		return len(d.Path) == l
	}
}

// WherePathDepthGt filters Differences - if Difference path is greater than l
func WherePathDepthGt(l int) DifferenceFilterFunc {
	return func(d Difference) bool {
		return len(d.Path) > l
	}
}

// WherePathDepthLt filters Differences - if Difference path is less than l
func WherePathDepthLt(l int) DifferenceFilterFunc {
	return func(d Difference) bool {
		return len(d.Path) < l
	}
}

// WhereDiffType filters Differences - if Difference type is dt
func WhereDiffType(dt DiffType) DifferenceFilterFunc {
	return func(d Difference) bool {
		return d.Type == dt
	}
}
