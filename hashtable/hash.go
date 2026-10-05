package hashtable

const (
	// fnvOffset64 and fnvPrime64 are the two parameters of the 64-bit FNV-1a
	// specification, which is the whole of what HashString has to implement.
	fnvOffset64 = 14695981039346656037
	fnvPrime64  = 1099511628211
)

// HashString returns the 64-bit FNV-1a hash of s. O(len(s)) time, O(1) space.
func HashString(s string) uint64 {
	h := uint64(fnvOffset64)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= fnvPrime64
	}
	return h
}

// HashInt returns a mixed hash of n, spreading strided keys across buckets rather than back onto each other. O(1) time, O(1) space.
func HashInt(n int) uint64 {
	x := uint64(n)
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}
