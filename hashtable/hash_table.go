// Package hashtable implements hash tables from first principles, by separate chaining and by open addressing.
package hashtable

import (
	"fmt"
	"strings"
)

const (
	// initialBuckets is the bucket count New starts with; every growth doubles it.
	initialBuckets = 8
	// maxLoadChained is the load factor above which HashTable grows. Chaining
	// tolerates a fuller table than probing does, because a collision costs one more
	// link rather than a longer walk over unrelated keys.
	maxLoadChained = 0.75
)

// HashTable is a hash table that resolves collisions by separate chaining: each
// bucket holds a chain of the entries whose keys hashed into it.
//
// It must be made with New. Alone among the structures in this repo the zero value
// is not usable, because a table cannot hash anything without a hash function and Go
// offers no way to hash an arbitrary comparable K. So the caller supplies one —
// HashString and HashInt in this package are two — and it becomes part of the value,
// the same bargain the other doc comments in this repo describe for an ordering:
// when the language cannot supply the operation, the caller does.
//
// A key is present at most once. Put overwrites the value of a key already there and
// reports false, reporting true only when it added one. Values are never compared
// and may repeat freely.
//
// New entries go at the head of their bucket's chain, so the insert itself is O(1).
// Scanning the chain is the real cost, and it makes Get, Put and Delete O(1) on
// average and O(n) when every key lands in the same bucket. That is why the quality
// of the hash function is closer to a correctness concern than a tuning detail: a
// constant hash leaves this type a linked list with extra steps.
//
// Put grows the table when adding an entry would push the load factor above
// maxLoadChained: the bucket count doubles and every entry is rehashed into the new
// buckets. Rehashing is the step that is easy to skip and hard to notice — a table
// whose chains were copied across rather than rehashed still answers correctly for
// every key that happened not to move. Delete never shrinks the table.
//
// Iteration order is unspecified. Keys and Values walk the buckets in index order,
// which is stable for a given hash function and bucket count and changes the moment
// either one does, so nothing should be read into it.
//
// A zero V is a legitimate value and a zero K a legitimate key, so Get reports
// presence through a separate bool — never by returning the zero value alone.
type HashTable[K comparable, V any] struct {
	buckets []*entry[K, V]
	size    int
	hash    func(K) uint64
}

// New returns an empty table that hashes its keys with hash. O(1) time, O(1) space.
func New[K comparable, V any](hash func(K) uint64) *HashTable[K, V] {
	return &HashTable[K, V]{
		buckets: make([]*entry[K, V], initialBuckets),
		hash:    hash,
	}
}

// Len returns the number of keys in the table. O(1) time, O(1) space.
func (t *HashTable[K, V]) Len() int {
	return t.size
}

// Put stores value under key, reporting false if it overwrote an existing key; it grows the table when the load factor demands it. O(1) average time, O(n) worst case, O(1) space.
func (t *HashTable[K, V]) Put(key K, value V) bool {
	if e := t.find(key); e != nil {
		e.value = value
		return false
	}
	if float64(t.size+1)/float64(len(t.buckets)) > maxLoadChained {
		t.grow()
	}
	i := t.index(key)
	t.buckets[i] = &entry[K, V]{key: key, value: value, next: t.buckets[i]}
	t.size++
	return true
}

// Get returns the value stored under key, comma-ok. O(1) average time, O(n) worst case, O(1) space.
func (t *HashTable[K, V]) Get(key K) (V, bool) {
	if e := t.find(key); e != nil {
		return e.value, true
	}
	var zero V
	return zero, false
}

// Contains reports whether key is in the table. O(1) average time, O(n) worst case, O(1) space.
func (t *HashTable[K, V]) Contains(key K) bool {
	return t.find(key) != nil
}

// Delete removes key, reporting whether it was there to remove. O(1) average time, O(n) worst case, O(1) space.
func (t *HashTable[K, V]) Delete(key K) bool {
	for link := &t.buckets[t.index(key)]; *link != nil; link = &(*link).next {
		if e := *link; e.key == key {
			*link = e.next
			e.next = nil
			t.size--
			return true
		}
	}
	return false
}

// Keys returns every key in unspecified order, empty and non-nil for an empty table. O(n) time, O(n) space.
func (t *HashTable[K, V]) Keys() []K {
	keys := make([]K, 0, t.size)
	for _, head := range t.buckets {
		for e := head; e != nil; e = e.next {
			keys = append(keys, e.key)
		}
	}
	return keys
}

// Values returns every value in unspecified order, with repeats, empty and non-nil for an empty table. O(n) time, O(n) space.
func (t *HashTable[K, V]) Values() []V {
	values := make([]V, 0, t.size)
	for _, head := range t.buckets {
		for e := head; e != nil; e = e.next {
			values = append(values, e.value)
		}
	}
	return values
}

// String renders the buckets in index order as "0:[] 1:[b=2 a=1]". O(n) time, O(n) space.
func (t *HashTable[K, V]) String() string {
	var b strings.Builder
	for i, head := range t.buckets {
		if i > 0 {
			b.WriteByte(' ')
		}
		fmt.Fprintf(&b, "%d:[", i)
		for e := head; e != nil; e = e.next {
			if e != head {
				b.WriteByte(' ')
			}
			fmt.Fprintf(&b, "%v=%v", e.key, e.value)
		}
		b.WriteByte(']')
	}
	return b.String()
}

// PrintTable writes String followed by a newline to standard output. O(n) time, O(n) space.
func (t *HashTable[K, V]) PrintTable() {
	fmt.Println(t.String())
}

// index returns the bucket key hashes to. O(1) time, O(1) space.
func (t *HashTable[K, V]) index(key K) int {
	return int(t.hash(key) % uint64(len(t.buckets)))
}

// find returns the entry holding key, or nil. O(1) average time, O(n) worst case, O(1) space.
func (t *HashTable[K, V]) find(key K) *entry[K, V] {
	for e := t.buckets[t.index(key)]; e != nil; e = e.next {
		if e.key == key {
			return e
		}
	}
	return nil
}

// grow doubles the bucket count and rehashes every entry into the new buckets. O(n) time, O(n) space.
func (t *HashTable[K, V]) grow() {
	old := t.buckets
	t.buckets = make([]*entry[K, V], 2*len(old))
	for _, head := range old {
		for e := head; e != nil; {
			next := e.next
			i := t.index(e.key)
			e.next = t.buckets[i]
			t.buckets[i] = e
			e = next
		}
	}
}
