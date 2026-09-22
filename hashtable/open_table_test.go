// White-box tests for the open-addressed table.
//
//	go test ./hashtable/ -run TestOpenTable -v      # this type
//	go test ./hashtable/ -run TestOpenTablePut -v   # one method
//
// Test names are prefixed with the type because two tables share this package and
// Go test function names are package-scoped; captureStdout, the pair type, idHash,
// zeroHash and sorted all live in hash_table_test.go, where one declaration serves
// both suites.
//
// With idHash and eight slots, key k comes home to slot k % 8 and probes forward
// from there, so 1, 9 and 17 land in slots 1, 2 and 3 in that order and the whole
// layout of a fixture is known before the first method is called. That is what
// makes the probe assertions below possible: they check where an entry sits, not
// merely that it can be found.
//
// assertOpenTable's sharpest check is reachability. For every live key it walks the
// probe sequence from that key's home slot and requires the key to turn up before
// any empty slot does, because an empty slot is where a search gives up. That single
// check is what catches the defining bug of open addressing: a Delete that clears
// its slot instead of leaving a tombstone looks perfect from the outside - the key
// is gone, the size is right, the table renders correctly - and has quietly cut
// every probe sequence that ran through that slot, stranding keys that are still
// there. Only a lookup of some unrelated key fails, and only sometimes.
//
// The helper also requires empty and deleted slots to hold a zero key and a zero
// value, since a tombstone needs neither and keeping them pins the objects for as
// long as the table lives.
//
// Three deliberate exceptions, each noted again above the test itself:
//   - TestOpenTableShape guards the shape of the structure rather than a method.
//   - TestOpenTablePrintTable delegates to String, because printing the rendered
//     table is the behavior under test.
//   - ExampleOpenTable exercises NewOpen, Put, Delete, Len and Get together,
//     because a runnable example is by definition an integration.
package hashtable

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"
)

// buildOpenTable returns a table of the given slot count holding pairs, placing
// each one by probing forward from its home slot exactly as Put is specified to,
// but without calling Put, so every test in this file is independent of the
// implementation and the methods can be written in any order. It refuses to build a
// fixture past the occupancy the table is allowed to reach, since a fixture that
// breaks the invariant under test is a bug in the test.
func buildOpenTable[K comparable, V any](hash func(K) uint64, slots int, pairs ...pair[K, V]) *OpenTable[K, V] {
	if slots <= 0 {
		panic("buildOpenTable: a table has at least one slot")
	}
	if float64(len(pairs))/float64(slots) > maxLoadOpen {
		panic("buildOpenTable: fixture is over the occupancy the table is allowed to reach")
	}
	t := &OpenTable[K, V]{slots: make([]slot[K, V], slots), hash: hash}
	for _, p := range pairs {
		home := int(hash(p.key) % uint64(slots))
		for step := 0; step < slots; step++ {
			i := (home + step) % slots
			if t.slots[i].state == slotFilled {
				if t.slots[i].key == p.key {
					panic("buildOpenTable: the same key twice")
				}
				continue
			}
			t.slots[i] = slot[K, V]{key: p.key, value: p.value, state: slotFilled}
			t.size++
			break
		}
	}
	return t
}

// entomb removes key the way Delete is specified to - the slot marked deleted with
// its key and value zeroed - without calling Delete, so a fixture can start out
// with tombstones already in it.
func entomb[K comparable, V any](t *OpenTable[K, V], key K) *OpenTable[K, V] {
	n := len(t.slots)
	home := int(t.hash(key) % uint64(n))
	for step := 0; step < n; step++ {
		i := (home + step) % n
		if t.slots[i].state == slotFilled && t.slots[i].key == key {
			t.slots[i] = slot[K, V]{state: slotDeleted}
			t.size--
			t.dead++
			return t
		}
		if t.slots[i].state == slotEmpty {
			break
		}
	}
	panic("entomb: the key is not in the table")
}

// liveAt reports the key in slot i, for the tests that assert a layout rather than
// a set of contents.
func liveAt[K comparable, V any](t *OpenTable[K, V], i int) (K, bool) {
	if t.slots[i].state != slotFilled {
		var zero K
		return zero, false
	}
	return t.slots[i].key, true
}

// assertOpenTable checks every structural invariant at once: the contents, the two
// counters, that no key is stored twice, that every live key is reachable by
// probing from its home slot, that occupancy is inside its bound, and that every
// slot which is not filled has been cleared. It reads the fields directly rather
// than calling Len, so it never borrows a method to judge another. Call it after
// every mutation, and after every rejected one too.
func assertOpenTable[K comparable, V comparable](t *testing.T, o *OpenTable[K, V], want map[K]V) {
	t.Helper()

	n := len(o.slots)
	if n == 0 {
		t.Fatalf("the table has no slots, want at least one")
	}

	got := make(map[K]V, len(want))
	live, dead := 0, 0
	var zeroK K
	var zeroV V

	for i, s := range o.slots {
		switch s.state {
		case slotFilled:
			live++
			if _, dup := got[s.key]; dup {
				t.Errorf("key %v is stored twice", s.key)
			}
			got[s.key] = s.value
		case slotDeleted:
			dead++
			if s.key != zeroK || s.value != zeroV {
				t.Errorf("tombstone in slot %d still holds %v=%v, want both cleared - a deleted entry stays alive as long as the table does", i, s.key, s.value)
			}
		case slotEmpty:
			if s.key != zeroK || s.value != zeroV {
				t.Errorf("empty slot %d holds %v=%v, want both cleared", i, s.key, s.value)
			}
		}
	}

	if !maps.Equal(got, want) {
		t.Errorf("entries = %v, want %v", got, want)
	}
	if o.size != len(want) {
		t.Errorf("size field = %d, want %d", o.size, len(want))
	}
	if o.size != live {
		t.Errorf("size field = %d, but %d slots are filled", o.size, live)
	}
	if o.dead != dead {
		t.Errorf("dead field = %d, but %d slots are tombstones", o.dead, dead)
	}
	if load := float64(live+dead) / float64(n); load > maxLoadOpen {
		t.Errorf("occupancy = %.3f over %d slots (%d live, %d dead), want no more than %.2f - the table should have grown", load, n, live, dead, maxLoadOpen)
	}

	// Reachability: from its home slot, a key must turn up before any empty slot,
	// because an empty slot is where a search stops.
	for j, s := range o.slots {
		if s.state != slotFilled {
			continue
		}
		home := int(o.hash(s.key) % uint64(n))
		for step := 0; step <= n; step++ {
			if step == n {
				t.Errorf("key %v in slot %d was not reached by probing the whole table from slot %d", s.key, j, home)
				break
			}
			i := (home + step) % n
			if i == j {
				break
			}
			if o.slots[i].state == slotEmpty {
				t.Errorf("key %v sits in slot %d but the probe from its home slot %d stops at the empty slot %d - the key is unreachable, which is what clearing a slot instead of leaving a tombstone leaves behind", s.key, j, home, i)
				break
			}
		}
	}
}

// TestOpenTableShape pins the shape of the structure rather than any one method.
// Entries live in the slot array itself, so there is no next pointer anywhere; the
// dead counter is here because tombstones have to drive growth alongside size, and
// the three slot states are the distinction the whole design turns on.
func TestOpenTableShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  reflect.Type
		want []string
	}{
		{"table", reflect.TypeOf(OpenTable[int, int]{}), []string{"slots", "size", "dead", "hash"}},
		{"slot", reflect.TypeOf(slot[int, int]{}), []string{"key", "value", "state"}},
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

	t.Run("the three states are distinct", func(t *testing.T) {
		if slotEmpty == slotFilled || slotEmpty == slotDeleted || slotFilled == slotDeleted {
			t.Error("slotEmpty, slotFilled and slotDeleted must differ - collapsing a tombstone into either of the others is the bug the type exists to avoid")
		}
	})

	// The zero slot has to read as empty, because a freshly made slice of slots is
	// all zeroes and nothing initialises it.
	t.Run("the zero slot is empty", func(t *testing.T) {
		var s slot[int, int]
		if s.state != slotEmpty {
			t.Errorf("the zero slot's state = %d, want slotEmpty (%d)", s.state, slotEmpty)
		}
	})
}

// TestOpenTableEmpty is the case most implementations get wrong, so every method
// gets a subtest against a freshly built empty table.
func TestOpenTableEmpty(t *testing.T) {
	t.Run("Len", func(t *testing.T) {
		if got := buildOpenTable[int, int](idHash, 8).Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
	})
	t.Run("Put", func(t *testing.T) {
		o := buildOpenTable[int, int](idHash, 8)
		if got := o.Put(1, 10); !got {
			t.Errorf("Put(1, 10) = false, want true - the key is new")
		}
		assertOpenTable(t, o, map[int]int{1: 10})
	})
	t.Run("Get", func(t *testing.T) {
		o := buildOpenTable[int, int](idHash, 8)
		got, ok := o.Get(1)
		if ok {
			t.Errorf("Get(1) ok = true, want false")
		}
		if got != 0 {
			t.Errorf("Get(1) = %d, want the zero value", got)
		}
		assertOpenTable(t, o, nil)
	})
	t.Run("Contains", func(t *testing.T) {
		if got := buildOpenTable[int, int](idHash, 8).Contains(1); got {
			t.Errorf("Contains(1) = true, want false")
		}
	})
	t.Run("Delete", func(t *testing.T) {
		o := buildOpenTable[int, int](idHash, 8)
		if got := o.Delete(1); got {
			t.Errorf("Delete(1) = true, want false")
		}
		assertOpenTable(t, o, nil)
	})
	t.Run("Keys", func(t *testing.T) {
		got := buildOpenTable[int, int](idHash, 8).Keys()
		if got == nil {
			t.Fatal("Keys() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("Keys() = %v, want empty", got)
		}
	})
	t.Run("Values", func(t *testing.T) {
		got := buildOpenTable[int, int](idHash, 8).Values()
		if got == nil {
			t.Fatal("Values() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("Values() = %v, want empty", got)
		}
	})
	t.Run("String", func(t *testing.T) {
		want := "0:- 1:- 2:- 3:-"
		if got := buildOpenTable[int, int](idHash, 4).String(); got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	})
	// A table that holds nothing but tombstones is a second kind of empty, and the
	// one where a search for an absent key can still go wrong.
	t.Run("nothing but tombstones", func(t *testing.T) {
		o := entomb(buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 1)
		if got := o.Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
		if got := o.Contains(1); got {
			t.Errorf("Contains(1) = true, want false")
		}
		if _, ok := o.Get(1); ok {
			t.Errorf("Get(1) ok = true, want false")
		}
		if got := o.Delete(1); got {
			t.Errorf("Delete(1) = true, want false")
		}
		assertOpenTable(t, o, nil)
	})
}

func TestOpenTableNewOpen(t *testing.T) {
	o := NewOpen[int, string](idHash)
	if o == nil {
		t.Fatal("NewOpen() = nil, want an empty table")
	}
	if len(o.slots) != initialSlots {
		t.Errorf("NewOpen() made %d slots, want %d", len(o.slots), initialSlots)
	}
	if o.hash == nil {
		t.Fatal("NewOpen() left the hash function nil, want the one it was given")
	}
	if got, want := o.hash(42), idHash(42); got != want {
		t.Errorf("the stored hash returned %d for 42, want %d - it is not the function NewOpen was given", got, want)
	}
	assertOpenTable(t, o, nil)
}

func TestOpenTableLen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *OpenTable[int, int]
		want  int
	}{
		{"empty", buildOpenTable[int, int](idHash, 8), 0},
		{"one", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 1},
		{"two in their own slots", buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}), 2},
		{"two sharing a home slot", buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 2},
		{"tombstones are not counted", entomb(buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 1), 1},
		{"the zero key counts", buildOpenTable(idHash, 8, pair[int, int]{0, 0}), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.table.Len(); got != tc.want {
				t.Errorf("Len() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestOpenTablePut(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *OpenTable[int, int]
		key   int
		value int
		want  bool
		after map[int]int
	}{
		{"into its home slot", buildOpenTable[int, int](idHash, 8), 1, 10, true, map[int]int{1: 10}},
		{"probing past one taken slot", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 9, 90, true, map[int]int{1: 10, 9: 90}},
		{"probing past two taken slots", buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 17, 170, true, map[int]int{1: 10, 9: 90, 17: 170}},
		{"overwriting a key in its home slot", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 1, 11, false, map[int]int{1: 11}},
		{"overwriting a key found by probing", buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 9, 99, false, map[int]int{1: 10, 9: 99}},
		{"wrapping past the end of the table", buildOpenTable(idHash, 8, pair[int, int]{7, 70}), 15, 150, true, map[int]int{7: 70, 15: 150}},
		{"every key colliding", buildOpenTable(zeroHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}), 3, 30, true, map[int]int{1: 10, 2: 20, 3: 30}},
		{"the zero key", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 0, 99, true, map[int]int{1: 10, 0: 99}},
		{"the zero value", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 2, 0, true, map[int]int{1: 10, 2: 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.table.Put(tc.key, tc.value); got != tc.want {
				t.Errorf("Put(%d, %d) = %v, want %v", tc.key, tc.value, got, tc.want)
			}
			assertOpenTable(t, tc.table, tc.after)
		})
	}

	t.Run("lands in the slot the probe reaches", func(t *testing.T) {
		o := buildOpenTable(idHash, 8, pair[int, int]{1, 10})
		o.Put(9, 90)
		if key, ok := liveAt(o, 2); !ok || key != 9 {
			t.Errorf("slot 2 holds %v (live %v), want key 9 - its home slot 1 is taken, so it belongs in the next one", key, ok)
		}
	})

	t.Run("wraps to the front rather than running off the end", func(t *testing.T) {
		o := buildOpenTable(idHash, 8, pair[int, int]{7, 70})
		o.Put(15, 150)
		if key, ok := liveAt(o, 0); !ok || key != 15 {
			t.Errorf("slot 0 holds %v (live %v), want key 15 - the probe from slot 7 wraps to the front", key, ok)
		}
	})

	// Reusing a tombstone is the whole reason Put is harder than it looks.
	t.Run("reuses a tombstone", func(t *testing.T) {
		o := entomb(buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 1)
		if got := o.Put(17, 170); !got {
			t.Fatalf("Put(17, 170) = false, want true")
		}
		if key, ok := liveAt(o, 1); !ok || key != 17 {
			t.Errorf("slot 1 holds %v (live %v), want key 17 - the tombstone there is free to reuse", key, ok)
		}
		if o.dead != 0 {
			t.Errorf("dead = %d, want 0 - reusing the tombstone consumes it", o.dead)
		}
		assertOpenTable(t, o, map[int]int{9: 90, 17: 170})
	})

	// The trap inside the trap: the first tombstone is only free once the rest of
	// the sequence has been checked. Settling into it early stores 9 twice.
	t.Run("updates a key that lies past a tombstone", func(t *testing.T) {
		o := entomb(buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 1)
		if got := o.Put(9, 99); got {
			t.Errorf("Put(9, 99) = true, want false - key 9 is already in the table, two slots along")
		}
		if _, ok := liveAt(o, 1); ok {
			t.Errorf("slot 1 is filled, want the tombstone left alone - key 9 was already stored further along")
		}
		if o.dead != 1 {
			t.Errorf("dead = %d, want 1", o.dead)
		}
		assertOpenTable(t, o, map[int]int{9: 99})
	})

	t.Run("grows when occupancy would be exceeded", func(t *testing.T) {
		o := buildOpenTable(idHash, 8,
			pair[int, int]{1, 10}, pair[int, int]{9, 90}, pair[int, int]{2, 20}, pair[int, int]{3, 30})
		if got := o.Put(4, 40); !got {
			t.Fatalf("Put(4, 40) = false, want true")
		}
		if len(o.slots) != 16 {
			t.Errorf("the table has %d slots, want 16 - the fifth entry takes occupancy past %.2f", len(o.slots), maxLoadOpen)
		}
		assertOpenTable(t, o, map[int]int{1: 10, 9: 90, 2: 20, 3: 30, 4: 40})
		// 9 hashed to slot 1 in a table of 8 and hashes to slot 9 in a table of 16,
		// so a growth that copies the slots across rather than rehashing them
		// strands it where assertOpenTable will find it.
		if key, ok := liveAt(o, 9); !ok || key != 9 {
			t.Errorf("slot 9 holds %v (live %v), want key 9 after the rehash", key, ok)
		}
	})

	t.Run("does not grow one entry too early", func(t *testing.T) {
		o := buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}, pair[int, int]{3, 30})
		o.Put(4, 40)
		if len(o.slots) != 8 {
			t.Errorf("the table has %d slots, want 8 - four entries in eight slots is exactly %.2f, which is not over the limit", len(o.slots), maxLoadOpen)
		}
		assertOpenTable(t, o, map[int]int{1: 10, 2: 20, 3: 30, 4: 40})
	})

	// Tombstones count toward occupancy, and growth is the only thing that clears
	// them. A table that grows on size alone fills with tombstones and probes like
	// a table a hundred times its size.
	t.Run("tombstones count toward growth and are cleared by it", func(t *testing.T) {
		o := buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}, pair[int, int]{3, 30})
		o = entomb(o, 2)
		o = entomb(o, 3)
		// One live entry, two tombstones: occupancy three of eight.
		o.Put(4, 40)
		if len(o.slots) != 8 {
			t.Fatalf("the table has %d slots, want 8 - four of eight is not over the limit", len(o.slots))
		}
		if got := o.Put(5, 50); !got {
			t.Fatalf("Put(5, 50) = false, want true")
		}
		if len(o.slots) != 16 {
			t.Errorf("the table has %d slots, want 16 - two tombstones and three live entries take occupancy past %.2f", len(o.slots), maxLoadOpen)
		}
		if o.dead != 0 {
			t.Errorf("dead = %d, want 0 - growth rehashes the live entries and drops the tombstones", o.dead)
		}
		assertOpenTable(t, o, map[int]int{1: 10, 4: 40, 5: 50})
	})

	t.Run("grows repeatedly and keeps every key reachable", func(t *testing.T) {
		o := NewOpen[int, int](idHash)
		want := map[int]int{}
		for i := 0; i < 200; i++ {
			key := i * 7
			if got := o.Put(key, i); !got {
				t.Fatalf("Put(%d, %d) = false, want true", key, i)
			}
			want[key] = i
			assertOpenTable(t, o, want)
		}
		if len(o.slots) < 512 {
			t.Errorf("the table has %d slots for %d keys, want at least 512", len(o.slots), len(want))
		}
	})

	t.Run("every key colliding into one probe sequence", func(t *testing.T) {
		o := NewOpen[int, int](zeroHash)
		want := map[int]int{}
		for i := 0; i < 50; i++ {
			o.Put(i, i*10)
			want[i] = i * 10
			assertOpenTable(t, o, want)
		}
	})
}

func TestOpenTableGet(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *OpenTable[int, int]
		key   int
		want  int
		ok    bool
	}{
		{"in its home slot", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 1, 10, true},
		{"one probe along", buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 9, 90, true},
		{"two probes along", buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}, pair[int, int]{17, 170}), 17, 170, true},
		{"wrapped past the end", buildOpenTable(idHash, 8, pair[int, int]{7, 70}, pair[int, int]{15, 150}), 15, 150, true},
		{"absent, home slot empty", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 2, 0, false},
		{"absent, home slot taken", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 9, 0, false},
		{"absent past the end of a run", buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 17, 0, false},
		{"the zero key", buildOpenTable(idHash, 8, pair[int, int]{0, 99}), 0, 99, true},
		{"absent when every key collides", buildOpenTable(zeroHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}), 3, 0, false},
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

	// The reason tombstones exist, stated as a test: 9 is still in the table and
	// still has to be found, with a deleted slot sitting between it and its home.
	t.Run("probes straight through a tombstone", func(t *testing.T) {
		o := entomb(buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 1)
		got, ok := o.Get(9)
		if !ok {
			t.Errorf("Get(9) ok = false, want true - the tombstone in slot 1 must not end the search")
		}
		if got != 90 {
			t.Errorf("Get(9) = %d, want 90", got)
		}
	})

	t.Run("stops at an empty slot rather than scanning the table", func(t *testing.T) {
		o := buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90})
		if _, ok := o.Get(4); ok {
			t.Errorf("Get(4) ok = true, want false - slot 4 is empty and nothing lies beyond it")
		}
	})

	// The Go trap: a stored zero is indistinguishable from a missing key unless the
	// bool is read.
	t.Run("a stored zero value is present", func(t *testing.T) {
		o := buildOpenTable(idHash, 8, pair[int, int]{1, 0})
		got, ok := o.Get(1)
		if !ok {
			t.Errorf("Get(1) ok = false, want true - the value is zero, the key is not missing")
		}
		if got != 0 {
			t.Errorf("Get(1) = %d, want 0", got)
		}
	})

	t.Run("does not mutate the table", func(t *testing.T) {
		o := buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90})
		o.Get(1)
		o.Get(9)
		o.Get(17)
		assertOpenTable(t, o, map[int]int{1: 10, 9: 90})
	})
}

func TestOpenTableContains(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *OpenTable[int, int]
		key   int
		want  bool
	}{
		{"in its home slot", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 1, true},
		{"one probe along", buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 9, true},
		{"past a tombstone", entomb(buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 1), 9, true},
		{"absent, home slot empty", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 2, false},
		{"absent, home slot taken", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 9, false},
		{"deleted", entomb(buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 1), 1, false},
		{"the zero key", buildOpenTable(idHash, 8, pair[int, int]{0, 99}), 0, true},
		{"a key whose value is zero", buildOpenTable(idHash, 8, pair[int, int]{1, 0}), 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.table.Contains(tc.key); got != tc.want {
				t.Errorf("Contains(%d) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}
}

func TestOpenTableDelete(t *testing.T) {
	run := func() *OpenTable[int, int] {
		return buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}, pair[int, int]{17, 170})
	}

	for _, tc := range []struct {
		name  string
		table *OpenTable[int, int]
		key   int
		want  bool
		after map[int]int
	}{
		{"the start of a run", run(), 1, true, map[int]int{9: 90, 17: 170}},
		{"the middle of a run", run(), 9, true, map[int]int{1: 10, 17: 170}},
		{"the end of a run", run(), 17, true, map[int]int{1: 10, 9: 90}},
		{"the only entry", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), 1, true, map[int]int{}},
		{"a wrapped entry", buildOpenTable(idHash, 8, pair[int, int]{7, 70}, pair[int, int]{15, 150}), 15, true, map[int]int{7: 70}},
		{"absent, home slot empty", run(), 4, false, map[int]int{1: 10, 9: 90, 17: 170}},
		{"absent, home slot taken", run(), 25, false, map[int]int{1: 10, 9: 90, 17: 170}},
		{"the zero key", buildOpenTable(idHash, 8, pair[int, int]{0, 99}, pair[int, int]{1, 10}), 0, true, map[int]int{1: 10}},
		{"a key whose value is zero", buildOpenTable(idHash, 8, pair[int, int]{1, 0}), 1, true, map[int]int{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.table.Delete(tc.key); got != tc.want {
				t.Errorf("Delete(%d) = %v, want %v", tc.key, got, tc.want)
			}
			assertOpenTable(t, tc.table, tc.after)
		})
	}

	// The defining case. Clearing slot 1 outright satisfies everything a caller can
	// see about key 1 and cuts 9 and 17 off from their home slot.
	t.Run("leaves a tombstone so later keys stay reachable", func(t *testing.T) {
		o := run()
		if got := o.Delete(1); !got {
			t.Fatalf("Delete(1) = false, want true")
		}
		if o.slots[1].state != slotDeleted {
			t.Errorf("slot 1 is in state %d, want slotDeleted (%d) - clearing it strands 9 and 17 behind a gap their search stops at", o.slots[1].state, slotDeleted)
		}
		if o.dead != 1 {
			t.Errorf("dead = %d, want 1", o.dead)
		}
		assertOpenTable(t, o, map[int]int{9: 90, 17: 170})
	})

	t.Run("clears the key and value it tombstones", func(t *testing.T) {
		o := run()
		o.Delete(9)
		if got := o.slots[2]; got.key != 0 || got.value != 0 {
			t.Errorf("the tombstone in slot 2 holds %v=%v, want both cleared - a deleted entry has no business outliving its slot", got.key, got.value)
		}
	})

	t.Run("twice reports false the second time", func(t *testing.T) {
		o := run()
		if got := o.Delete(9); !got {
			t.Fatalf("Delete(9) = false, want true")
		}
		if got := o.Delete(9); got {
			t.Errorf("Delete(9) again = true, want false")
		}
		assertOpenTable(t, o, map[int]int{1: 10, 17: 170})
	})

	t.Run("drains the whole run", func(t *testing.T) {
		o := run()
		want := map[int]int{1: 10, 9: 90, 17: 170}
		for _, key := range []int{9, 1, 17} {
			if got := o.Delete(key); !got {
				t.Fatalf("Delete(%d) = false, want true", key)
			}
			delete(want, key)
			assertOpenTable(t, o, want)
		}
		if o.dead != 3 {
			t.Errorf("dead = %d, want 3 - every removal leaves a tombstone behind", o.dead)
		}
	})

	t.Run("delete then put the same key back", func(t *testing.T) {
		o := run()
		o.Delete(9)
		assertOpenTable(t, o, map[int]int{1: 10, 17: 170})
		if got := o.Put(9, 99); !got {
			t.Errorf("Put(9, 99) = true expected for a key that was deleted, got false")
		}
		assertOpenTable(t, o, map[int]int{1: 10, 9: 99, 17: 170})
	})
}

func TestOpenTableKeys(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *OpenTable[int, int]
		want  []int
	}{
		{"one", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), []int{1}},
		{"spread across slots", buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{3, 30}), []int{1, 3}},
		{"in one run", buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}, pair[int, int]{17, 170}), []int{1, 9, 17}},
		{"skipping tombstones", entomb(buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 1), []int{9}},
		{"including the zero key", buildOpenTable(idHash, 8, pair[int, int]{0, 99}, pair[int, int]{1, 10}), []int{0, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sorted(tc.table.Keys()); !slices.Equal(got, tc.want) {
				t.Errorf("sorted Keys() = %v, want %v", got, tc.want)
			}
		})
	}

	// A tombstone holds a zero key, so a Keys that walks the slots without checking
	// the state returns a phantom zero key for every entry ever deleted.
	t.Run("a tombstone is not a zero key", func(t *testing.T) {
		o := entomb(buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 20}), 1)
		got := o.Keys()
		if len(got) != 1 {
			t.Fatalf("Keys() = %v, want exactly one key", got)
		}
		if got[0] != 2 {
			t.Errorf("Keys() = %v, want [2] - the cleared tombstone must not read as key 0", got)
		}
	})
}

func TestOpenTableValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *OpenTable[int, int]
		want  []int
	}{
		{"one", buildOpenTable(idHash, 8, pair[int, int]{1, 10}), []int{10}},
		{"spread across slots", buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{3, 30}), []int{10, 30}},
		{"repeats are kept", buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{2, 10}), []int{10, 10}},
		{"skipping tombstones", entomb(buildOpenTable(idHash, 8, pair[int, int]{1, 10}, pair[int, int]{9, 90}), 1), []int{90}},
		{"including a zero value", buildOpenTable(idHash, 8, pair[int, int]{1, 0}, pair[int, int]{2, 20}), []int{0, 20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sorted(tc.table.Values()); !slices.Equal(got, tc.want) {
				t.Errorf("sorted Values() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestOpenTableString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *OpenTable[int, int]
		want  string
	}{
		{"empty", buildOpenTable[int, int](idHash, 4), "0:- 1:- 2:- 3:-"},
		{"one entry", buildOpenTable(idHash, 4, pair[int, int]{1, 10}), "0:- 1:1=10 2:- 3:-"},
		{"a run of two", buildOpenTable(idHash, 4, pair[int, int]{1, 10}, pair[int, int]{5, 50}), "0:- 1:1=10 2:5=50 3:-"},
		{"a tombstone", entomb(buildOpenTable(idHash, 4, pair[int, int]{1, 10}, pair[int, int]{5, 50}), 1), "0:- 1:x 2:5=50 3:-"},
		{"slot zero", buildOpenTable(idHash, 4, pair[int, int]{0, 99}), "0:0=99 1:- 2:- 3:-"},
		{"wrapped", buildOpenTable(idHash, 4, pair[int, int]{3, 30}, pair[int, int]{7, 70}), "0:7=70 1:- 2:- 3:3=30"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.table.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestOpenTablePrintTable delegates to String by design - printing the rendered
// table is the behavior under test - so it goes red if String is broken.
func TestOpenTablePrintTable(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table *OpenTable[int, int]
		want  string
	}{
		{"empty", buildOpenTable[int, int](idHash, 4), "0:- 1:- 2:- 3:-\n"},
		{"one entry", buildOpenTable(idHash, 4, pair[int, int]{1, 10}), "0:- 1:1=10 2:- 3:-\n"},
		{"a tombstone", entomb(buildOpenTable(idHash, 4, pair[int, int]{1, 10}), 1), "0:- 1:x 2:- 3:-\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := captureStdout(t, tc.table.PrintTable); got != tc.want {
				t.Errorf("PrintTable() wrote %q, want %q", got, tc.want)
			}
		})
	}
}

func ExampleOpenTable() {
	t := NewOpen[int, string](HashInt)
	t.Put(1, "one")
	t.Put(2, "two")
	fmt.Println(t.Len())

	fmt.Println(t.Delete(1), t.Len())

	word, ok := t.Get(2)
	fmt.Println(word, ok)
	// Output:
	// 2
	// true 1
	// two true
}
