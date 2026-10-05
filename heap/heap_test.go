// White-box tests for the binary heap and KLargest.
//
//	go test ./heap/ -v                  # everything in the package
//	go test ./heap/ -run TestHeapPop -v # one method
//
// The oracle never asks the heap what it holds. assertHeap reads items directly and
// checks the heap property at every index, so a Push that sifts the wrong way or a
// Pop that swaps with the wrong child fails here even when the root happens to come
// out right. It compares contents as a sorted multiset rather than a layout, because
// the layout a correct heap produces depends on its sift details and the contract
// promises only the invariant - the one exception is ToSlice and String, whose
// contract is the layout of a fixture that never moves.
//
// buildHeap writes the backing array exactly as given and refuses one that is not a
// heap, so every test is independent of Push and Heapify and the methods can be
// written in any order.
//
// Deliberate exceptions, each noted again above the test itself:
//   - TestHeapShape guards the shape of the structure rather than a method.
//   - TestHeapPrintHeap delegates to String, because printing the rendered heap is
//     the behavior under test.
//   - TestKLargest calls only KLargest, but a KLargest built on Heap - the point of
//     the exercise - stays red until New, Push, Pop and Len are written.
//   - ExampleHeap exercises NewMin, Push, Pop and Len together, because a runnable
//     example is by definition an integration.
package heap

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"testing"
)

// intLess and intGreater are the two comparators most fixtures use, written out
// rather than borrowed from NewMin and NewMax so no fixture depends on a stub.
func intLess(a, b int) bool    { return a < b }
func intGreater(a, b int) bool { return a > b }

// task is a struct with an ordering on one field, the priority-queue case a
// comparator exists for.
type task struct {
	name     string
	priority int
}

func byPriority(a, b task) bool { return a.priority < b.priority }

// violation returns the first index whose element is ordered before its parent, or
// -1 if items is a heap under less.
func violation[T any](items []T, less func(a, b T) bool) int {
	for i := 1; i < len(items); i++ {
		if less(items[i], items[(i-1)/2]) {
			return i
		}
	}
	return -1
}

// buildHeap returns a heap whose backing array is exactly items, without calling
// Push or Heapify. It refuses a fixture that is not a heap, since a fixture that
// violates the invariant under test is a bug in the test.
func buildHeap[T any](less func(a, b T) bool, items ...T) *Heap[T] {
	if i := violation(items, less); i >= 0 {
		panic(fmt.Sprintf("buildHeap: fixture %v is not a heap at index %d", items, i))
	}
	return &Heap[T]{items: slices.Clone(items), less: less}
}

// sortedBy returns a copy of values sorted so that less elements come first.
func sortedBy[T any](values []T, less func(a, b T) bool) []T {
	out := slices.Clone(values)
	slices.SortStableFunc(out, func(a, b T) int {
		switch {
		case less(a, b):
			return -1
		case less(b, a):
			return 1
		}
		return 0
	})
	return out
}

// assertHeap checks every structural invariant at once: the heap property at every
// index, that the contents are want as a multiset, and that the cell above the
// length holds the zero value. It reads items directly, never Len or ToSlice. Call
// it after every mutation, and after every rejected one too.
func assertHeap[T comparable](t *testing.T, h *Heap[T], want []T) {
	t.Helper()

	if h.less == nil {
		t.Fatalf("less is nil, want the comparator the heap was made with")
	}
	if i := violation(h.items, h.less); i >= 0 {
		t.Errorf("items = %v breaks the heap property at index %d: %v is ordered before its parent %v", h.items, i, h.items[i], h.items[(i-1)/2])
	}
	if got, w := sortedBy(h.items, h.less), sortedBy(want, h.less); !slices.Equal(got, w) {
		t.Errorf("contents = %v, want %v (sorted, order within a heap is not part of the contract)", got, w)
	}
	var zero T
	if n := len(h.items); cap(h.items) > n && h.items[:n+1][n] != zero {
		t.Errorf("the cell above the last element holds %v, want the zero value - Pop left the popped element reachable", h.items[:n+1][n])
	}
}

// captureStdout redirects os.Stdout for the duration of f and returns what was
// written.
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

// TestHeapShape pins the shape of the structure rather than any one method. The
// backing store is a slice and the ordering is a field, which is why the zero value
// is not usable.
func TestHeapShape(t *testing.T) {
	var got []string
	for _, f := range reflect.VisibleFields(reflect.TypeOf(Heap[int]{})) {
		got = append(got, f.Name)
	}
	if want := []string{"items", "less"}; !slices.Equal(got, want) {
		t.Errorf("Heap fields = %v, want %v", got, want)
	}
}

// TestHeapEmpty is the case most implementations get wrong, so every method gets a
// subtest against a freshly built empty heap.
func TestHeapEmpty(t *testing.T) {
	t.Run("Len", func(t *testing.T) {
		if got := buildHeap(intLess).Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
	})
	t.Run("IsEmpty", func(t *testing.T) {
		if got := buildHeap(intLess).IsEmpty(); !got {
			t.Errorf("IsEmpty() = false, want true")
		}
	})
	t.Run("Push", func(t *testing.T) {
		h := buildHeap(intLess)
		h.Push(10)
		assertHeap(t, h, []int{10})
	})
	t.Run("Pop", func(t *testing.T) {
		h := buildHeap(intLess)
		got, err := h.Pop()
		if !errors.Is(err, ErrEmptyHeap) {
			t.Errorf("Pop() error = %v, want ErrEmptyHeap", err)
		}
		if got != 0 {
			t.Errorf("Pop() = %v, want the zero value on error", got)
		}
		assertHeap(t, h, nil)
	})
	t.Run("Peek", func(t *testing.T) {
		h := buildHeap(intLess)
		got, err := h.Peek()
		if !errors.Is(err, ErrEmptyHeap) {
			t.Errorf("Peek() error = %v, want ErrEmptyHeap", err)
		}
		if got != 0 {
			t.Errorf("Peek() = %v, want the zero value on error", got)
		}
		assertHeap(t, h, nil)
	})
	t.Run("Heapify", func(t *testing.T) {
		assertHeap(t, Heapify(nil, intLess), nil)
	})
	t.Run("ToSlice", func(t *testing.T) {
		got := buildHeap(intLess).ToSlice()
		if got == nil || len(got) != 0 {
			t.Errorf("ToSlice() = %#v, want an empty non-nil slice", got)
		}
	})
	t.Run("String", func(t *testing.T) {
		if got := buildHeap(intLess).String(); got != "[]" {
			t.Errorf("String() = %q, want %q", got, "[]")
		}
	})
	t.Run("KLargest", func(t *testing.T) {
		got := KLargest([]int{}, 3)
		if got == nil || len(got) != 0 {
			t.Errorf("KLargest([], 3) = %#v, want an empty non-nil slice", got)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("is empty", func(t *testing.T) {
		h := New(intLess)
		if h == nil {
			t.Fatal("New() = nil, want a heap")
		}
		assertHeap(t, h, nil)
	})
	t.Run("keeps the comparator it was given", func(t *testing.T) {
		h := New(intGreater)
		if h.less == nil || !h.less(2, 1) || h.less(1, 2) {
			t.Error("less does not order like the comparator passed to New")
		}
	})
}

func TestNewMin(t *testing.T) {
	h := NewMin[int]()
	if h == nil {
		t.Fatal("NewMin() = nil, want a heap")
	}
	assertHeap(t, h, nil)
	for _, tc := range []struct {
		a, b int
		want bool
	}{
		{1, 2, true},
		{2, 1, false},
		{1, 1, false},
		{-5, 0, true},
	} {
		if got := h.less(tc.a, tc.b); got != tc.want {
			t.Errorf("less(%d, %d) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
	if s := NewMin[string](); !s.less("apple", "banana") {
		t.Error(`less("apple", "banana") = false, want true for a min-heap of strings`)
	}
}

func TestNewMax(t *testing.T) {
	h := NewMax[int]()
	if h == nil {
		t.Fatal("NewMax() = nil, want a heap")
	}
	assertHeap(t, h, nil)
	for _, tc := range []struct {
		a, b int
		want bool
	}{
		{2, 1, true},
		{1, 2, false},
		{1, 1, false},
		{0, -5, true},
	} {
		if got := h.less(tc.a, tc.b); got != tc.want {
			t.Errorf("less(%d, %d) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestHeapify(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []int
		less   func(a, b int) bool
	}{
		{"one", []int{7}, intLess},
		{"two in order", []int{1, 2}, intLess},
		{"two out of order", []int{2, 1}, intLess},
		{"ascending", []int{1, 2, 3, 4, 5, 6, 7}, intLess},
		{"descending", []int{9, 8, 7, 6, 5, 4, 3, 2, 1}, intLess},
		{"shuffled", []int{5, 3, 8, 1, 9, 2, 7, 4, 6, 0}, intLess},
		{"duplicates", []int{4, 1, 4, 1, 4, 1}, intLess},
		{"all equal", []int{3, 3, 3, 3}, intLess},
		{"zero values", []int{0, 5, 0, -1}, intLess},
		{"max-heap", []int{5, 3, 8, 1, 9, 2, 7}, intGreater},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := slices.Clone(tc.values)
			h := Heapify(tc.values, tc.less)
			if h == nil {
				t.Fatal("Heapify() = nil, want a heap")
			}
			assertHeap(t, h, before)
			if !slices.Equal(tc.values, before) {
				t.Errorf("Heapify reordered its input to %v, want it left as %v", tc.values, before)
			}
		})
	}

	t.Run("does not alias its input", func(t *testing.T) {
		values := []int{3, 1, 2}
		h := Heapify(values, intLess)
		values[0], values[1], values[2] = 100, 200, 300
		assertHeap(t, h, []int{1, 2, 3})
	})

	// Sifting down from the last parent to the root is O(n); a heapify that only
	// sifts the root, or walks parents root-first, leaves a deep violation behind.
	t.Run("restores order deep in the tree", func(t *testing.T) {
		values := make([]int, 31)
		for i := range values {
			values[i] = len(values) - i
		}
		h := Heapify(values, intLess)
		assertHeap(t, h, values)
	})
}

func TestHeapLen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		items []int
	}{
		{"one", []int{1}},
		{"two", []int{1, 2}},
		{"many", []int{1, 2, 3, 4, 5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildHeap(intLess, tc.items...).Len(); got != len(tc.items) {
				t.Errorf("Len() = %d, want %d", got, len(tc.items))
			}
		})
	}
}

func TestHeapIsEmpty(t *testing.T) {
	if got := buildHeap(intLess, 1).IsEmpty(); got {
		t.Error("IsEmpty() = true on one element, want false")
	}
	if got := buildHeap(intLess, 0).IsEmpty(); got {
		t.Error("IsEmpty() = true on a single zero value, want false - zero is an element")
	}
}

func TestHeapPush(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		push  int
	}{
		{"onto one, larger", []int{5}, 9},
		{"onto one, smaller", []int{5}, 1},
		{"new root", []int{2, 4, 3, 8, 5}, 1},
		{"stays a leaf", []int{2, 4, 3, 8, 5}, 9},
		{"rises one level", []int{1, 4, 3, 8, 5}, 2},
		{"rises to the root from a deep leaf", []int{1, 2, 3, 4, 5, 6, 7, 8}, 0},
		{"equal to the root", []int{1, 2, 3}, 1},
		{"equal to its parent", []int{1, 5, 3}, 5},
		{"zero value", []int{1, 2, 3}, 0},
		{"negative", []int{-1, 2, 3}, -7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := buildHeap(intLess, tc.start...)
			h.Push(tc.push)
			assertHeap(t, h, append(slices.Clone(tc.start), tc.push))
		})
	}

	t.Run("ascending run", func(t *testing.T) {
		h := buildHeap(intLess)
		var want []int
		for i := 1; i <= 20; i++ {
			h.Push(i)
			want = append(want, i)
			assertHeap(t, h, want)
		}
	})

	t.Run("descending run", func(t *testing.T) {
		h := buildHeap(intLess)
		var want []int
		for i := 20; i >= 1; i-- {
			h.Push(i)
			want = append(want, i)
			assertHeap(t, h, want)
		}
	})

	t.Run("max-heap", func(t *testing.T) {
		h := buildHeap(intGreater, 9, 5, 7)
		h.Push(10)
		assertHeap(t, h, []int{9, 5, 7, 10})
		if h.items[0] != 10 {
			t.Errorf("root = %d, want 10 in a max-heap", h.items[0])
		}
	})

	t.Run("structs by priority", func(t *testing.T) {
		h := buildHeap(byPriority, task{"write", 2}, task{"test", 3})
		h.Push(task{"plan", 1})
		assertHeap(t, h, []task{{"write", 2}, {"test", 3}, {"plan", 1}})
		if h.items[0].name != "plan" {
			t.Errorf("root = %v, want the priority-1 task", h.items[0])
		}
	})
}

func TestHeapPop(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
	}{
		{"only element", []int{7}, 7},
		{"two", []int{1, 2}, 1},
		{"three", []int{1, 3, 2}, 1},
		{"smaller child on the right", []int{1, 5, 2, 6, 7, 3, 4}, 1},
		{"smaller child on the left", []int{1, 2, 5, 3, 4, 6, 7}, 1},
		{"last leaf sinks to the bottom", []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, 1},
		{"duplicate roots", []int{1, 1, 1, 2}, 1},
		{"zero value", []int{0, 1, 2}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := buildHeap(intLess, tc.start...)
			got, err := h.Pop()
			if err != nil {
				t.Fatalf("Pop() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Pop() = %d, want %d", got, tc.want)
			}
			rest := slices.Clone(tc.start)
			rest = slices.Delete(rest, slices.Index(rest, tc.want), slices.Index(rest, tc.want)+1)
			assertHeap(t, h, rest)
		})
	}

	// Draining comes out sorted and checks the invariant after every single pop,
	// which is where a sift-down that stops one level early shows.
	for _, tc := range []struct {
		name  string
		start []int
		less  func(a, b int) bool
	}{
		{"drains in ascending order", []int{1, 3, 2, 7, 4, 5, 6, 8, 9, 10}, intLess},
		{"drains with duplicates", []int{1, 1, 2, 2, 3, 3, 3}, intLess},
		{"drains a max-heap descending", []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}, intGreater},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := buildHeap(tc.less, tc.start...)
			want := sortedBy(tc.start, tc.less)
			for i, w := range want {
				got, err := h.Pop()
				if err != nil {
					t.Fatalf("Pop() %d error = %v, want nil", i, err)
				}
				if got != w {
					t.Fatalf("Pop() %d = %d, want %d", i, got, w)
				}
				assertHeap(t, h, want[i+1:])
			}
			if _, err := h.Pop(); !errors.Is(err, ErrEmptyHeap) {
				t.Errorf("Pop() on the drained heap error = %v, want ErrEmptyHeap", err)
			}
		})
	}

	t.Run("zeroes the vacated slot", func(t *testing.T) {
		a, b, c := 1, 2, 3
		h := buildHeap(func(x, y *int) bool { return *x < *y }, &a, &b, &c)
		if _, err := h.Pop(); err != nil {
			t.Fatalf("Pop() error = %v, want nil", err)
		}
		assertHeap(t, h, []*int{&b, &c})
	})
}

func TestHeapPeek(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		less  func(a, b int) bool
		want  int
	}{
		{"one", []int{7}, intLess, 7},
		{"min-heap", []int{1, 3, 2}, intLess, 1},
		{"max-heap", []int{9, 3, 7}, intGreater, 9},
		{"zero value", []int{0, 1}, intLess, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := buildHeap(tc.less, tc.start...)
			got, err := h.Peek()
			if err != nil {
				t.Fatalf("Peek() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Peek() = %d, want %d", got, tc.want)
			}
			if !slices.Equal(h.items, tc.start) {
				t.Errorf("items = %v after Peek, want %v untouched", h.items, tc.start)
			}
		})
	}
}

func TestHeapToSlice(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
	}{
		{"one", []int{7}},
		{"two", []int{1, 2}},
		{"level order, not sorted", []int{1, 5, 2, 6, 7, 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := buildHeap(intLess, tc.start...)
			if got := h.ToSlice(); !slices.Equal(got, tc.start) {
				t.Errorf("ToSlice() = %v, want %v", got, tc.start)
			}
		})
	}

	t.Run("returns a copy", func(t *testing.T) {
		h := buildHeap(intLess, 1, 2, 3)
		got := h.ToSlice()
		got[0] = 99
		assertHeap(t, h, []int{1, 2, 3})
		if h.items[0] != 1 {
			t.Errorf("writing to the result changed the heap's root to %d", h.items[0])
		}
	})
}

func TestHeapString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"one", []int{7}, "[7]"},
		{"level order", []int{1, 3, 2}, "[1 3 2]"},
		{"zero value", []int{0, 4}, "[0 4]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildHeap(intLess, tc.start...).String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestHeapPrintHeap delegates to String by design - printing the rendered heap is
// the behavior under test - so it goes red if String is broken.
func TestHeapPrintHeap(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "[]\n"},
		{"many", []int{1, 3, 2}, "[1 3 2]\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := captureStdout(t, buildHeap(intLess, tc.start...).PrintHeap); got != tc.want {
				t.Errorf("PrintHeap() wrote %q, want %q", got, tc.want)
			}
		})
	}
}

// TestKLargest calls only KLargest, but an implementation built on Heap - the
// intended one - stays red until New, Push, Pop and Len are written.
func TestKLargest(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []int
		k      int
		want   []int
	}{
		{"k is zero", []int{3, 1, 2}, 0, []int{}},
		{"k is negative", []int{3, 1, 2}, -1, []int{}},
		{"k is one", []int{3, 9, 2}, 1, []int{9}},
		{"k is two", []int{5, 1, 9, 3, 7}, 2, []int{9, 7}},
		{"k is the length", []int{2, 3, 1}, 3, []int{3, 2, 1}},
		{"k past the length", []int{2, 3, 1}, 10, []int{3, 2, 1}},
		{"one value", []int{4}, 1, []int{4}},
		{"duplicates count separately", []int{5, 5, 5, 1}, 3, []int{5, 5, 5}},
		{"a duplicate straddles the cut", []int{1, 4, 4, 2}, 2, []int{4, 4}},
		{"negatives", []int{-3, -1, -2}, 2, []int{-1, -2}},
		{"zero is a value", []int{0, -1, -2}, 1, []int{0}},
		{"largest arrive last", []int{1, 2, 3, 4, 5, 6}, 3, []int{6, 5, 4}},
		{"largest arrive first", []int{6, 5, 4, 3, 2, 1}, 3, []int{6, 5, 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := slices.Clone(tc.values)
			got := KLargest(tc.values, tc.k)
			if got == nil {
				t.Fatalf("KLargest(%v, %d) = nil, want a non-nil slice", tc.values, tc.k)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("KLargest(%v, %d) = %v, want %v", before, tc.k, got, tc.want)
			}
			if !slices.Equal(tc.values, before) {
				t.Errorf("KLargest reordered its input to %v, want it left as %v", tc.values, before)
			}
		})
	}

	t.Run("strings", func(t *testing.T) {
		got := KLargest([]string{"pear", "apple", "plum", "fig"}, 2)
		if want := []string{"plum", "pear"}; !slices.Equal(got, want) {
			t.Errorf("KLargest = %v, want %v", got, want)
		}
	})

	t.Run("agrees with sorting", func(t *testing.T) {
		values := make([]int, 200)
		for i := range values {
			values[i] = (i * 37) % 101
		}
		want := slices.Clone(values)
		slices.SortFunc(want, func(a, b int) int { return cmp.Compare(b, a) })
		for _, k := range []int{1, 5, 50, 199, 200} {
			if got := KLargest(values, k); !slices.Equal(got, want[:k]) {
				t.Errorf("KLargest(k=%d) = %v, want %v", k, got, want[:k])
			}
		}
	})
}

func ExampleHeap() {
	h := NewMin[int]()
	for _, v := range []int{5, 2, 8, 1} {
		h.Push(v)
	}
	var drained []int
	for h.Len() > 0 {
		v, _ := h.Pop()
		drained = append(drained, v)
	}
	fmt.Println(drained)
	fmt.Println(KLargest([]int{5, 2, 8, 1}, 2))
	// Output:
	// [1 2 5 8]
	// [8 5]
}
