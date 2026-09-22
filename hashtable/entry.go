package hashtable

// entry is one link in a bucket's chain: a key, its value, and the next entry that
// hashed into the same bucket.
type entry[K comparable, V any] struct {
	key   K
	value V
	next  *entry[K, V]
}
