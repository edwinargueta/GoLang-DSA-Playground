package hashtable

const (
	// fnvOffset64 and fnvPrime64 are the two parameters of the 64-bit FNV-1a
	// specification, which is the whole of what HashString has to implement.
	fnvOffset64 = 14695981039346656037
	fnvPrime64  = 1099511628211
)

// HashString returns the 64-bit FNV-1a hash of s. O(len(s)) time, O(1) space.
func HashString(s string) uint64 {
	panic("not implemented")
}

// HashInt returns a mixed hash of n, spreading strided keys across buckets rather than back onto each other. O(1) time, O(1) space.
func HashInt(n int) uint64 {
	panic("not implemented")
}
