package heap

import "cmp"

// KLargest returns the k largest of values largest first, duplicates counted separately, values untouched; k <= 0 gives an empty non-nil slice, k > len(values) gives all. O(n log k) time, O(k) space.
func KLargest[T cmp.Ordered](values []T, k int) []T {
	panic("not implemented")
}
