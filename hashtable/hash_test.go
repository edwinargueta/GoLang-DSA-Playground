// Tests for the package's hash functions.
//
//	go test ./hashtable/ -run TestHash -v        # both functions
//	go test ./hashtable/ -run TestHashString -v  # one of them
//
// The two functions are tested very differently on purpose.
//
// HashString has a specification. FNV-1a is a published algorithm with published
// test vectors, so the exact output for a given input is the behavior, not an
// implementation detail, and the table below pins it. Getting a different number
// for "foobar" does not mean a different valid choice was made; it means the
// function is not FNV-1a.
//
// HashInt has no such specification, and the doc comment deliberately asks for a
// property rather than an algorithm: strided keys must not stay strided. So the
// tests here assert properties - determinism, distinct outputs for distinct inputs,
// and that keys a power of two apart do not all land in one bucket - and leave the
// choice of mixer entirely open. Anything that satisfies them is a correct answer.
//
// The distribution tests use no randomness. They run the same fixed inputs every
// time, so a pass means a pass rather than a lucky seed.
//
// ExampleHashString is the one deliberate exception: it prints a vector already
// covered by the table, because a runnable example doubles as the documentation of
// which algorithm this is.
package hashtable

import (
	"fmt"
	"testing"
)

func TestHashString(t *testing.T) {
	// The published FNV-1a 64-bit vectors.
	for _, tc := range []struct {
		in   string
		want uint64
	}{
		{"", 14695981039346656037},
		{"a", 12638187200555641996},
		{"b", 12638190499090526629},
		{"c", 12638189399578898418},
		{"foobar", 9625390261332436968},
		{"hello", 11831194018420276491},
	} {
		t.Run(fmt.Sprintf("vector %q", tc.in), func(t *testing.T) {
			if got := HashString(tc.in); got != tc.want {
				t.Errorf("HashString(%q) = %d (%#x), want %d (%#x)", tc.in, got, got, tc.want, tc.want)
			}
		})
	}

	t.Run("the empty string is the offset basis untouched", func(t *testing.T) {
		if got := HashString(""); got != fnvOffset64 {
			t.Errorf("HashString(%q) = %d, want the offset basis %d - there is nothing to fold in", "", got, uint64(fnvOffset64))
		}
	})

	t.Run("is deterministic", func(t *testing.T) {
		for _, s := range []string{"", "a", "hello", "a longer string with spaces"} {
			if first, second := HashString(s), HashString(s); first != second {
				t.Errorf("HashString(%q) returned %d then %d, want the same value twice", s, first, second)
			}
		}
	})

	t.Run("order of bytes matters", func(t *testing.T) {
		if HashString("ab") == HashString("ba") {
			t.Error(`HashString("ab") == HashString("ba"), want different - a hash that only sums its bytes loses the order`)
		}
	})

	t.Run("nearby strings do not collide", func(t *testing.T) {
		seen := map[uint64]string{}
		for _, s := range []string{"a", "b", "c", "aa", "ab", "ba", "key1", "key2", "key10", "Key1", ""} {
			h := HashString(s)
			if other, dup := seen[h]; dup {
				t.Errorf("HashString(%q) == HashString(%q) == %d, want different values", s, other, h)
			}
			seen[h] = s
		}
	})

	// Not a measure of cryptographic quality - just enough to catch a function
	// whose output barely moves, which is what a missing multiply leaves behind.
	t.Run("spreads a run of similar keys across buckets", func(t *testing.T) {
		const (
			buckets = 16
			keys    = 1000
		)
		counts := make([]int, buckets)
		for i := 0; i < keys; i++ {
			counts[HashString(fmt.Sprintf("key%d", i))%buckets]++
		}
		average := keys / buckets
		for i, count := range counts {
			if count == 0 {
				t.Errorf("bucket %d got none of the %d keys, want roughly %d", i, keys, average)
			}
			if count > 3*average {
				t.Errorf("bucket %d got %d of the %d keys, want roughly %d", i, count, keys, average)
			}
		}
	})
}

func TestHashInt(t *testing.T) {
	t.Run("is deterministic", func(t *testing.T) {
		for _, n := range []int{0, 1, -1, 42, 1 << 40, -(1 << 40)} {
			if first, second := HashInt(n), HashInt(n); first != second {
				t.Errorf("HashInt(%d) returned %d then %d, want the same value twice", n, first, second)
			}
		}
	})

	t.Run("distinct keys give distinct hashes", func(t *testing.T) {
		seen := map[uint64]int{}
		for n := -500; n < 500; n++ {
			h := HashInt(n)
			if other, dup := seen[h]; dup {
				t.Errorf("HashInt(%d) == HashInt(%d) == %d, want different values", n, other, h)
			}
			seen[h] = n
		}
	})

	t.Run("the zero key hashes like any other", func(t *testing.T) {
		if HashInt(0) == HashInt(1) {
			t.Error("HashInt(0) == HashInt(1), want different values")
		}
	})

	t.Run("negative keys work", func(t *testing.T) {
		if HashInt(-1) == HashInt(1) {
			t.Error("HashInt(-1) == HashInt(1), want different values - the sign is part of the key")
		}
	})

	// The property the doc comment asks for, and the reason the function exists at
	// all. Returning the key unchanged passes every test above and fails this one
	// outright: sixty-four keys sixty-four apart would all come home to one bucket.
	t.Run("spreads strided keys across buckets", func(t *testing.T) {
		const (
			buckets = 64
			stride  = 64
		)
		reached := map[uint64]bool{}
		for i := 0; i < buckets; i++ {
			reached[HashInt(i*stride)%buckets] = true
		}
		if len(reached) < 20 {
			t.Errorf("%d keys %d apart reached %d of %d buckets, want at least 20 - an unmixed hash sends every one of them to the same bucket", buckets, stride, len(reached), buckets)
		}
	})

	t.Run("spreads a run of consecutive keys across buckets", func(t *testing.T) {
		const (
			buckets = 16
			keys    = 1000
		)
		counts := make([]int, buckets)
		for i := 0; i < keys; i++ {
			counts[HashInt(i)%buckets]++
		}
		average := keys / buckets
		for i, count := range counts {
			if count == 0 {
				t.Errorf("bucket %d got none of the %d keys, want roughly %d", i, keys, average)
			}
			if count > 3*average {
				t.Errorf("bucket %d got %d of the %d keys, want roughly %d", i, count, keys, average)
			}
		}
	})
}

func ExampleHashString() {
	fmt.Printf("%#x\n", HashString("foobar"))
	// Output: 0x85944171f73967e8
}
