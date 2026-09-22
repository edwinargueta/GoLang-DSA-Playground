package hashtable

const (
	// initialSlots is the slot count NewOpen starts with; every growth doubles it.
	initialSlots = 8
	// maxLoadOpen is the occupancy — live entries plus tombstones — above which
	// OpenTable grows. It is lower than maxLoadChained because probing degrades
	// before the table is anywhere near full: at nine tenths occupancy a linear
	// probe reads dozens of unrelated keys to answer one lookup, and at full
	// occupancy a search for an absent key never terminates at all.
	maxLoadOpen = 0.5
)

// OpenTable is a hash table that resolves collisions by open addressing with linear
// probing: every entry lives in the slot array itself, and a key whose home slot is
// taken moves forward one slot at a time, wrapping past the end, until a free one
// turns up.
//
// It must be made with NewOpen, for the same reason HashTable must be made with New:
// the hash function is supplied by the caller and is part of the value.
//
// The whole of the difficulty is deletion. Clearing a slot outright would cut every
// probe sequence that runs through it, stranding keys that are still in the table
// behind a gap a search stops at. So Delete leaves a tombstone: a slot marked
// deleted, which a search probes straight through and an insert is free to reuse.
// Delete also zeroes the slot's key and value, since a tombstone needs neither and
// holding them keeps them alive for as long as the table lives.
//
// Tombstones are why occupancy, not size, drives growth. A table cycled through
// enough puts and deletes fills with them, and a probe walks tombstones as patiently
// as it walks live entries; a table holding three keys can probe like one holding
// three hundred. Growth is also the only thing that clears them: it rehashes the
// live entries into a fresh array of twice the slots and drops every tombstone on
// the floor. One consequence is worth seeing rather than fixing — a table churned
// hard grows even though its live size never does.
//
// Put has one subtlety that follows from all of this: it may only settle into the
// first tombstone it passed once it has probed the rest of the sequence and
// established the key is not already stored further along. Inserting into the
// tombstone immediately is the classic way to end up with the same key twice.
//
// A key is present at most once, Put reports false when it overwrote a key and true
// when it added one, and values may repeat. Iteration order is unspecified. A zero V
// is a legitimate value and a zero K a legitimate key, so Get reports presence
// through a separate bool — never by returning the zero value alone.
//
// Against HashTable: one allocation for the whole table instead of one per entry,
// and probing walks memory in order where a chain chases pointers, which is most of
// why open addressing is the faster of the two in practice. Against that, it is the
// one that needs tombstones, a stricter load factor, and twice the care.
type OpenTable[K comparable, V any] struct {
	slots []slot[K, V]
	size  int
	dead  int
	hash  func(K) uint64
}

// NewOpen returns an empty table that hashes its keys with hash. O(1) time, O(1) space.
func NewOpen[K comparable, V any](hash func(K) uint64) *OpenTable[K, V] {
	panic("not implemented")
}

// Len returns the number of live keys, not counting tombstones. O(1) time, O(1) space.
func (t *OpenTable[K, V]) Len() int {
	panic("not implemented")
}

// Put stores value under key, reporting false if it overwrote an existing key; it reuses a tombstone where it safely can and grows the table when occupancy demands it. O(1) average time, O(n) worst case, O(1) space.
func (t *OpenTable[K, V]) Put(key K, value V) bool {
	panic("not implemented")
}

// Get returns the value stored under key, comma-ok. O(1) average time, O(n) worst case, O(1) space.
func (t *OpenTable[K, V]) Get(key K) (V, bool) {
	panic("not implemented")
}

// Contains reports whether key is in the table. O(1) average time, O(n) worst case, O(1) space.
func (t *OpenTable[K, V]) Contains(key K) bool {
	panic("not implemented")
}

// Delete replaces key's entry with a tombstone, zeroing the slot's key and value, and reports whether the key was there. O(1) average time, O(n) worst case, O(1) space.
func (t *OpenTable[K, V]) Delete(key K) bool {
	panic("not implemented")
}

// Keys returns every live key in unspecified order, empty and non-nil for an empty table. O(n) time, O(n) space.
func (t *OpenTable[K, V]) Keys() []K {
	panic("not implemented")
}

// Values returns every live value in unspecified order, with repeats, empty and non-nil for an empty table. O(n) time, O(n) space.
func (t *OpenTable[K, V]) Values() []V {
	panic("not implemented")
}

// String renders the slots in index order as "0:- 1:a=1 2:x", with "-" for empty and "x" for a tombstone. O(n) time, O(n) space.
func (t *OpenTable[K, V]) String() string {
	panic("not implemented")
}

// PrintTable writes String followed by a newline to standard output. O(n) time, O(n) space.
func (t *OpenTable[K, V]) PrintTable() {
	panic("not implemented")
}
