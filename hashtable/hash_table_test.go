// White-box tests for the chained hash table.
//
//	go test ./hashtable/ -v                        # everything in the package
//	go test ./hashtable/ -run TestHashTablePut -v  # one method
//
// Two tables and two hash functions share this package and Go test function names
// are package-scoped, so every test here is prefixed with what it covers.
// captureStdout, the pair type, the fixed hash functions and sorted are declared
// here and used by open_table_test.go as well, since one declaration per package is
// all Go allows.
//
// The suites hash with idHash and zeroHash rather than with HashString, which is
// the point of a table that takes its hash function as an argument. idHash sends
// key k to bucket k % len(buckets), so a test can say exactly which bucket every
// key lands in and asserting the layout becomes possible at all; zeroHash sends
// every key to bucket zero, collapsing the table into a single chain and exercising
// the collision path on purpose rather than hoping to stumble into it.
//
// The contents oracle is a plain Go map. Checking the reimplementation against the
// thing being reimplemented is exactly what an oracle is for - it is an independent,
// trusted implementation - and it is the only place in the suite where that is
// allowed.
//
// assertTable's sharpest check is that every entry sits in the bucket its key
// hashes to now. A growth that copies chains into the new array without rehashing
// them answers Get correctly for every key that happens not to have moved and
// silently strands the rest, and neither the size counter nor a small table's
// contents reveal it.
//
// Three deliberate exceptions, each noted again above the test itself:
//   - TestHashTableShape guards the shape of the structure rather than a method.
//   - TestHashTablePrintTable delegates to String, because printing the rendered
//     table is the behavior under test.
//   - ExampleHashTable exercises New, Put, Get and Len together, because a runnable
//     example is by definition an integration.
package hashtable

import (
	"bytes"
	"cmp"
	"fmt"
	"io"
	"maps"
	"os"
	"reflect"
	"slices"
	"testing"
)

// maxEntries bounds every walking helper, so a chain wired into a cycle fails the
// suite instead of hanging it.
const maxEntries = 5000

// pair is one key and value, so a fixture can be written as a list rather than a
// map and keep its insertion order - which is what decides chain order.
type pair[K comparable, V any] struct {
	key   K
	value V
}

// idHash sends key k to bucket k % len(buckets), which is what lets these tests
// state a layout instead of discovering one. It is a deliberately terrible hash for
// real use and a perfect one for a test.
func idHash(k int) uint64 { return uint64(k) }

// zeroHash sends every key to bucket zero, collapsing a chained table into one
// chain and an open-addressed one into a single probe sequence from slot zero.
func zeroHash(int) uint64 { return 0 }

// sorted returns a copy in order, for the methods that promise no order at all.
func sorted[T cmp.Ordered](values []T) []T {
	out := slices.Clone(values)
	slices.Sort(out)
	return out
}

// buildTable returns a table of the given bucket count holding pairs, linking each
// entry into the bucket its key hashes to with the newest at the head - the layout
// Put is specified to produce - without calling Put, so every test in this file is
// independent of the implementation and the methods can be written in any order. It
// refuses to build a fixture that breaks the load factor, since a fixture that
// violates the invariant under test is a bug in the test.
func buildTable[K comparable, V any](hash func(K) uint64, buckets int, pairs ...pair[K, V]) *HashTable[K, V] {
	if buckets <= 0 {
		panic("buildTable: a table has at least one bucket")
	}
	if float64(len(pairs))/float64(buckets) > maxLoadChained {
		panic("buildTable: fixture is over the load factor the table is allowed to reach")
	}
	t := &HashTable[K, V]{buckets: make([]*entry[K, V], buckets), hash: hash}
	for _, p := range pairs {
		i := int(hash(p.key) % uint64(buckets))
		t.buckets[i] = &entry[K, V]{key: p.key, value: p.value, next: t.buckets[i]}
		t.size++
	}
	return t
}

// assertTable checks every structural invariant at once: the contents, the size
// counter, that no key is stored twice, that every entry sits in the bucket its key
// hashes to, and that the load factor is inside its bound. It reads t.size directly
// rather than calling Len, so it never borrows a method to judge another. Call it
// after every mutation, and after every rejected one too - a call that reports false
// must leave the table as it was apart from the value it overwrote.
func assertTable[K comparable, V comparable](t *testing.T, h *HashTable[K, V], want map[K]V) {
	t.Helper()

	if len(h.buckets) == 0 {
		t.Fatalf("the table has no buckets, want at least one")
	}

	got := make(map[K]V, len(want))
	count := 0
	for i, head := range h.buckets {
		for e := head; e != nil; e = e.next {
			if count++; count > maxEntries {
				panic("walked past the limit - a bucket chain has a cycle")
			}
			if _, dup := got[e.key]; dup {
				t.Errorf("key %v is stored twice", e.key)
			}
			got[e.key] = e.value
			if home := int(h.hash(e.key) % uint64(len(h.buckets))); home != i {
				t.Errorf("key %v sits in bucket %d but hashes to bucket %d - it is unreachable, which is what a growth that does not rehash leaves behind", e.key, i, home)
			}
		}
	}

	if !maps.Equal(got, want) {
		t.Errorf("entries = %v, want %v", got, want)
	}
	if h.size != len(want) {
		t.Errorf("size field = %d, want %d", h.size, len(want))
	}
	if load := float64(count) / float64(len(h.buckets)); load > maxLoadChained {
		t.Errorf("load factor = %.3f over %d buckets, want no more than %.2f - the table should have grown", load, len(h.buckets), maxLoadChained)
	}
}

// captureStdout redirects os.Stdout for the duration of f and returns what was
// written. Shared by both table suites in this package.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	f()

	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading captured stdout failed: %v", err)
	}
	return buf.String()
}

// TestHashTableShape pins the shape of the structure rather than any one method.
// The hash function is a field, which is the whole reason this type cannot have a
// usable zero value, and the entry carries a next pointer, which is what makes this
// chaining rather than probing.
func TestHashTableShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  reflect.Type
		want []string
	}{
		{"table", reflect.TypeOf(HashTable[int, int]{}), []string{"buckets", "size", "hash"}},
		{"entry", reflect.TypeOf(entry[int, int]{}), []string{"key", "value", "next"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, f := range reflect.VisibleFields(tc.typ) {
				got = append(got, f.Name)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("%s fields = %v, want %v", tc.typ.Name(), got, tc.want)
			}
		})
	}
}

// TestHashTableEmpty is the case most implementations get wrong, so every method
// gets a subtest against a freshly built empty table.
func TestHashTableEmpty(t *testing.T) {
	t.Run("Len", func(t *testing.T) {
		if got := buildTable[int, int](idHash, 8).Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
	})
	t.Run("Put", func(t *testing.T) {
		h := buildTable[int, int](idHash, 8)
		if got := h.Put(1, 10); !got {
			t.Errorf("Put(1, 10) = false, want true - the key is new")
		}
		assertTable(t, h, map[int]int{1: 10})
	})
	t.Run("Get", func(t *testing.T) {
		h := buildTable[int, int](idHash, 8)
		got, ok := h.Get(1)
		if ok {
			t.Errorf("Get(1) ok = true, want false")
		}
		if got != 0 {
			t.Errorf("Get(1) = %d, want the zero value", got)
		}
		assertTable(t, h, nil)
	})
	t.Run("Contains", func(t *testing.T) {
		if got := buildTable[int, int](idHash, 8).Contains(1); got {
			t.Errorf("Contains(1) = true, want false")
		}
	})
	t.Run("Delete", func(t *testing.T) {
		h := buildTable[int, int](idHash, 8)
		if got := h.Delete(1); got {
			t.Errorf("Delete(1) = true, want false")
		}
		assertTable(t, h, nil)
	})
	t.Run("Keys", func(t *testing.T) {
		got := buildTable[int, int](idHash, 8).Keys()
		if got == nil {
			t.Fatal("Keys() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("Keys() = %v, want empty", got)
		}
	})
	t.Run("Values", func(t *testing.T) {
		got := buildTable[int, int](idHash, 8).Values()
		if got == nil {
			t.Fatal("Values() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("Values() = %v, want empty", got)
		}
	})
	t.Run("String", func(t *testing.T) {
		want := "0:[] 1:[] 2:[] 3:[]"
		if got := buildTable[int, int](idHash, 4).String(); got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})
}

func TestHashTableNew(t *testing.T) {
	h := New[int, string](idHash)
	if h == nil {
		t.Fatal("New() = nil, want an empty table")
	}
	if len(h.buckets) != initialBuckets {
		t.Errorf("New() made %d buckets, want %d", len(h.buckets), initialBuckets)
	}
	if h.hash == nil {
		t.Fatal("New() left the hash function nil, want the one it was given")
	}
	if got, want := h.hash(42), idHash(42); got != want {
		t.Errorf("the stored hash returned %d for 42, want %d - it is not the function New was given", got, want)
	}
	assertTable(t, h, nil)
}

func TestHashTableLen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *HashTable[int, int]
		want  int
	}{
		{"empty", buildTable[int, int](idHash, 8), 0},
		{"one", buildTable(idHash, 8, pair[int, int]{1, 10}), 1},
		{"two in different buckets", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}), 2},
		{"two in one bucket", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 2},
		{"every key in one bucket", buildTable(zeroHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}, pair[int, int]{3, 30}), 3},
		{"the zero key counts", buildTable(idHash, 8, pair[int, int]{0, 0}), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.table.Len(); got != tc.want {
				t.Errorf("Len() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestHashTablePut(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *HashTable[int, int]
		key   int
		value int
		want  bool
		after map[int]int
	}{
		{"into an empty bucket", buildTable[int, int](idHash, 8), 1, 10, true, map[int]int{1: 10}},
		{"into an occupied bucket", buildTable(idHash, 8, pair[int, int]{1, 10}), 9, 90, true, map[int]int{1: 10, 9: 90}},
		{"onto the head of a chain", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 9, 99, false, map[int]int{1: 10, 9: 99}},
		{"onto the tail of a chain", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 1, 11, false, map[int]int{1: 11, 9: 90}},
		{"every key colliding", buildTable(zeroHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}), 3, 30, true, map[int]int{1: 10, 2: 20, 3: 30}},
		{"the zero key", buildTable(idHash, 8, pair[int, int]{1, 10}), 0, 99, true, map[int]int{1: 10, 0: 99}},
		{"the zero value", buildTable(idHash, 8, pair[int, int]{1, 10}), 2, 0, true, map[int]int{1: 10, 2: 0}},
		{"a repeated value under a new key", buildTable(idHash, 8, pair[int, int]{1, 10}), 2, 10, true, map[int]int{1: 10, 2: 10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.table.Put(tc.key, tc.value); got != tc.want {
				t.Errorf("Put(%d, %d) = %v, want %v", tc.key, tc.value, got, tc.want)
			}
			assertTable(t, tc.table, tc.after)
		})
	}

	t.Run("overwriting does not change the size", func(t *testing.T) {
		h := buildTable(idHash, 8, pair[int, int]{1, 10})
		for i := 0; i < 20; i++ {
			if got := h.Put(1, i); got {
				t.Fatalf("Put(1, %d) = true, want false - the key is already there", i)
			}
		}
		assertTable(t, h, map[int]int{1: 19})
		if len(h.buckets) != 8 {
			t.Errorf("the table grew to %d buckets while only overwriting, want 8", len(h.buckets))
		}
	})

	// Growth belongs to Put: nothing else in the API can trigger it. The keys here
	// are chosen so that k % 8 and k % 16 differ, because keys below the old bucket
	// count land in the same index either way and a growth that forgets to rehash
	// goes unnoticed on them.
	t.Run("grows when the load factor would be exceeded", func(t *testing.T) {
		h := buildTable(idHash, 8,
			pair[int, int]{1, 10}, pair[int, int]{9, 90}, pair[int, int]{17, 170},
			pair[int, int]{2, 20}, pair[int, int]{3, 30}, pair[int, int]{4, 40})
		assertTable(t, h, map[int]int{1: 10, 9: 90, 17: 170, 2: 20, 3: 30, 4: 40})

		if got := h.Put(5, 50); !got {
			t.Fatalf("Put(5, 50) = false, want true")
		}
		if len(h.buckets) != 16 {
			t.Errorf("the table has %d buckets, want 16 - the seventh key takes the load factor past %.2f", len(h.buckets), maxLoadChained)
		}
		assertTable(t, h, map[int]int{1: 10, 9: 90, 17: 170, 2: 20, 3: 30, 4: 40, 5: 50})
	})

	t.Run("does not grow one key too early", func(t *testing.T) {
		h := buildTable(idHash, 8,
			pair[int, int]{1, 10}, pair[int, int]{2, 20}, pair[int, int]{3, 30},
			pair[int, int]{4, 40}, pair[int, int]{5, 50})
		h.Put(6, 60)
		if len(h.buckets) != 8 {
			t.Errorf("the table has %d buckets, want 8 - six keys in eight buckets is exactly %.2f, which is not over the limit", len(h.buckets), maxLoadChained)
		}
		assertTable(t, h, map[int]int{1: 10, 2: 20, 3: 30, 4: 40, 5: 50, 6: 60})
	})

	t.Run("grows repeatedly and keeps every key reachable", func(t *testing.T) {
		h := New[int, int](idHash)
		want := map[int]int{}
		for i := 0; i < 200; i++ {
			key := i * 7
			if got := h.Put(key, i); !got {
				t.Fatalf("Put(%d, %d) = false, want true", key, i)
			}
			want[key] = i
			assertTable(t, h, want)
		}
		if len(h.buckets) < 256 {
			t.Errorf("the table has %d buckets for %d keys, want at least 256", len(h.buckets), len(want))
		}
	})

	// One chain the length of the table. Everything still has to work; it is only
	// the cost that collapses.
	t.Run("every key in one bucket", func(t *testing.T) {
		h := New[int, int](zeroHash)
		want := map[int]int{}
		for i := 0; i < 50; i++ {
			h.Put(i, i*10)
			want[i] = i * 10
			assertTable(t, h, want)
		}
	})
}

func TestHashTableGet(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *HashTable[int, int]
		key   int
		want  int
		ok    bool
	}{
		{"the only key", buildTable(idHash, 8, pair[int, int]{1, 10}), 1, 10, true},
		{"at the head of a chain", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 9, 90, true},
		{"at the tail of a chain", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 1, 10, true},
		{"in the middle of a chain", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}, pair[int, int]{17, 170}), 9, 90, true},
		{"absent from an empty bucket", buildTable(idHash, 8, pair[int, int]{1, 10}), 2, 0, false},
		{"absent from an occupied bucket", buildTable(idHash, 8, pair[int, int]{1, 10}), 9, 0, false},
		{"absent past the end of a chain", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 17, 0, false},
		{"the zero key", buildTable(idHash, 8, pair[int, int]{0, 99}), 0, 99, true},
		{"absent when every key collides", buildTable(zeroHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}), 3, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := tc.table.Get(tc.key)
			if ok != tc.ok {
				t.Errorf("Get(%d) ok = %v, want %v", tc.key, ok, tc.ok)
			}
			if got != tc.want {
				t.Errorf("Get(%d) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}

	// The Go trap: a stored zero is indistinguishable from a missing key unless the
	// bool is read, so a Get that reports presence by comparing against the zero
	// value passes everything above and fails here.
	t.Run("a stored zero value is present", func(t *testing.T) {
		h := buildTable(idHash, 8, pair[int, int]{1, 0})
		got, ok := h.Get(1)
		if !ok {
			t.Errorf("Get(1) ok = false, want true - the value is zero, the key is not missing")
		}
		if got != 0 {
			t.Errorf("Get(1) = %d, want 0", got)
		}
	})

	t.Run("does not mutate the table", func(t *testing.T) {
		h := buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90})
		h.Get(1)
		h.Get(9)
		h.Get(17)
		assertTable(t, h, map[int]int{1: 10, 9: 90})
	})
}

func TestHashTableContains(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *HashTable[int, int]
		key   int
		want  bool
	}{
		{"the only key", buildTable(idHash, 8, pair[int, int]{1, 10}), 1, true},
		{"at the head of a chain", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 9, true},
		{"at the tail of a chain", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 1, true},
		{"absent from an empty bucket", buildTable(idHash, 8, pair[int, int]{1, 10}), 2, false},
		{"absent from an occupied bucket", buildTable(idHash, 8, pair[int, int]{1, 10}), 9, false},
		{"the zero key", buildTable(idHash, 8, pair[int, int]{0, 99}), 0, true},
		{"a key whose value is zero", buildTable(idHash, 8, pair[int, int]{1, 0}), 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.table.Contains(tc.key); got != tc.want {
				t.Errorf("Contains(%d) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestHashTableDelete(t *testing.T) {
	// Chains are built newest first, so keys 1, 9 and 17 in that order leave bucket
	// one reading 17 -> 9 -> 1: the three positions Delete has to unlink.
	chain := func() *HashTable[int, int] {
		return buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}, pair[int, int]{17, 170})
	}

	for _, tc := range []struct {
		name  string
		table *HashTable[int, int]
		key   int
		want  bool
		after map[int]int
	}{
		{"the head of a chain", chain(), 17, true, map[int]int{1: 10, 9: 90}},
		{"the middle of a chain", chain(), 9, true, map[int]int{1: 10, 17: 170}},
		{"the tail of a chain", chain(), 1, true, map[int]int{9: 90, 17: 170}},
		{"the only entry in a bucket", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}), 2, true, map[int]int{1: 10}},
		{"the only entry in the table", buildTable(idHash, 8, pair[int, int]{1, 10}), 1, true, map[int]int{}},
		{"absent from an empty bucket", chain(), 3, false, map[int]int{1: 10, 9: 90, 17: 170}},
		{"absent from an occupied bucket", chain(), 25, false, map[int]int{1: 10, 9: 90, 17: 170}},
		{"the zero key", buildTable(idHash, 8, pair[int, int]{0, 99}, pair[int, int]{1, 10}), 0, true, map[int]int{1: 10}},
		{"a key whose value is zero", buildTable(idHash, 8, pair[int, int]{1, 0}), 1, true, map[int]int{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.table.Delete(tc.key); got != tc.want {
				t.Errorf("Delete(%d) = %v, want %v", tc.key, got, tc.want)
			}
			assertTable(t, tc.table, tc.after)
		})
	}

	t.Run("twice reports false the second time", func(t *testing.T) {
		h := chain()
		if got := h.Delete(9); !got {
			t.Fatalf("Delete(9) = false, want true")
		}
		if got := h.Delete(9); got {
			t.Errorf("Delete(9) again = true, want false")
		}
		assertTable(t, h, map[int]int{1: 10, 17: 170})
	})

	t.Run("drains the whole chain", func(t *testing.T) {
		h := chain()
		want := map[int]int{1: 10, 9: 90, 17: 170}
		for _, key := range []int{9, 17, 1} {
			if got := h.Delete(key); !got {
				t.Fatalf("Delete(%d) = false, want true", key)
			}
			delete(want, key)
			assertTable(t, h, want)
		}
		if h.buckets[1] != nil {
			t.Errorf("bucket 1 still holds a chain, want nil once it is empty")
		}
	})

	t.Run("never shrinks the table", func(t *testing.T) {
		h := New[int, int](idHash)
		for i := 0; i < 50; i++ {
			h.Put(i, i)
		}
		grown := len(h.buckets)
		for i := 0; i < 50; i++ {
			h.Delete(i)
		}
		assertTable(t, h, map[int]int{})
		if len(h.buckets) != grown {
			t.Errorf("the table has %d buckets after draining, want the %d it grew to - Delete does not shrink", len(h.buckets), grown)
		}
	})
}

func TestHashTableKeys(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *HashTable[int, int]
		want  []int
	}{
		{"one", buildTable(idHash, 8, pair[int, int]{1, 10}), []int{1}},
		{"spread across buckets", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}, pair[int, int]{3, 30}), []int{1, 2, 3}},
		{"sharing one bucket", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}, pair[int, int]{17, 170}), []int{1, 9, 17}},
		{"all colliding", buildTable(zeroHash, 8, pair[int, int]{5, 50}, pair[int, int]{6, 60}), []int{5, 6}},
		{"including the zero key", buildTable(idHash, 8, pair[int, int]{0, 99}, pair[int, int]{1, 10}), []int{0, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Sorted, because the table promises no order and a test that depended
			// on one would be asserting an implementation detail.
			if got := sorted(tc.table.Keys()); !slices.Equal(got, tc.want) {
				t.Errorf("sorted Keys() = %v, want %v", got, tc.want)
			}
		})
	}

	t.Run("holds every key exactly once", func(t *testing.T) {
		h := buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}, pair[int, int]{17, 170})
		got := h.Keys()
		if len(got) != 3 {
			t.Errorf("Keys() returned %d keys, want 3", len(got))
		}
		seen := map[int]bool{}
		for _, k := range got {
			if seen[k] {
				t.Errorf("Keys() returned %d twice", k)
			}
			seen[k] = true
		}
	})
}

func TestHashTableValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *HashTable[int, int]
		want  []int
	}{
		{"one", buildTable(idHash, 8, pair[int, int]{1, 10}), []int{10}},
		{"spread across buckets", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}), []int{10, 20}},
		{"sharing one bucket", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), []int{10, 90}},
		{"repeats are kept", buildTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 10}, pair[int, int]{3, 10}), []int{10, 10, 10}},
		{"including a zero value", buildTable(idHash, 8, pair[int, int]{1, 0}, pair[int, int]{2, 20}), []int{0, 20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sorted(tc.table.Values()); !slices.Equal(got, tc.want) {
				t.Errorf("sorted Values() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHashTableString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *HashTable[int, int]
		want  string
	}{
		{"empty", buildTable[int, int](idHash, 4), "0:[] 1:[] 2:[] 3:[]"},
		{"one entry", buildTable(idHash, 4, pair[int, int]{1, 10}), "0:[] 1:[1=10] 2:[] 3:[]"},
		{"newest at the head of its chain", buildTable(idHash, 4, pair[int, int]{1, 10}, pair[int, int]{5, 50}), "0:[] 1:[5=50 1=10] 2:[] 3:[]"},
		{"spread across buckets", buildTable(idHash, 4, pair[int, int]{1, 10}, pair[int, int]{2, 20}), "0:[] 1:[1=10] 2:[2=20] 3:[]"},
		{"bucket zero", buildTable(idHash, 4, pair[int, int]{0, 99}), "0:[0=99] 1:[] 2:[] 3:[]"},
		{"a table of two buckets", buildTable(idHash, 2, pair[int, int]{7, 70}), "0:[] 1:[7=70]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.table.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestHashTablePrintTable delegates to String by design - printing the rendered
// table is the behavior under test - so it goes red if String is broken.
func TestHashTablePrintTable(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *HashTable[int, int]
		want  string
	}{
		{"empty", buildTable[int, int](idHash, 4), "0:[] 1:[] 2:[] 3:[]\n"},
		{"one entry", buildTable(idHash, 4, pair[int, int]{1, 10}), "0:[] 1:[1=10] 2:[] 3:[]\n"},
		{"a chain", buildTable(idHash, 4, pair[int, int]{1, 10}, pair[int, int]{5, 50}), "0:[] 1:[5=50 1=10] 2:[] 3:[]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := captureStdout(t, tc.table.PrintTable); got != tc.want {
				t.Errorf("PrintTable() wrote %q, want %q", got, tc.want)
			}
		})
	}
}

func ExampleHashTable() {
	t := New[string, int](HashString)
	t.Put("apples", 3)
	t.Put("pears", 5)
	t.Put("apples", 4)

	count, ok := t.Get("apples")
	fmt.Println(count, ok, t.Len())

	fmt.Println(t.Delete("pears"), t.Contains("pears"))
	// Output:
	// 4 true 2
	// true false
}
