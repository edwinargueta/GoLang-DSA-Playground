// White-box tests for the doubly linked list.
//
//	go test ./doublylinkedlist/ -v              # the whole suite
//	go test ./doublylinkedlist/ -run TestPop -v # one method
//
// The list stores head, tail and size, and every node points both ways. This is the
// structure those pointers belong to - the singly linked list's suite carries the
// mirror of TestDoublyLinked, asserting their absence there.
//
// A stale prev is the characteristic bug here and it is invisible to a forward
// traversal, so assertIntact walks the chain from both ends and then compares node
// identity in both directions. Values alone are not enough: with duplicates a broken
// prev can still spell the right sequence backward.
//
// The file is package doublylinkedlist, not ..._test, so the oracle helpers below can
// read head, tail, next, prev and size while those fields stay unexported. Every
// assertion goes through those helpers: no method is ever used to verify another, so a
// red test names exactly one broken method.
//
// Three deliberate exceptions, each noted again above the test itself:
//   - TestDoublyLinked guards the shape of the structure rather than a method.
//   - TestPrintList delegates to String, because printing the rendered chain is the
//     behavior under test.
//   - ExampleDoublyLinkedList exercises New, Append, Prepend and PrintList together,
//     because a runnable example is by definition an integration.
package doublylinkedlist

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"testing"
)

// maxNodes bounds every walking helper, so a chain wired into a cycle fails the suite
// instead of hanging it.
const maxNodes = 1000

// walkForward collects values by following next from head, independent of any method.
func walkForward[T comparable](head *node[T], limit int) []T {
	var values []T
	for n := head; n != nil; n = n.next {
		if len(values) > limit {
			panic("walked past the limit - the forward chain has a cycle")
		}
		values = append(values, n.value)
	}
	return values
}

// walkBackward collects values by following prev from tail, independent of any method.
func walkBackward[T comparable](tail *node[T], limit int) []T {
	var values []T
	for n := tail; n != nil; n = n.prev {
		if len(values) > limit {
			panic("walked past the limit - the backward chain has a cycle")
		}
		values = append(values, n.value)
	}
	return values
}

// reversed returns a copy of values back to front, so assertIntact can state what the
// backward walk should produce without calling Reverse or ToSliceReverse.
func reversed[T comparable](values []T) []T {
	out := make([]T, len(values))
	for i, v := range values {
		out[len(values)-1-i] = v
	}
	return out
}

// build wires nodes by hand rather than calling Append, so every test in this file is
// independent of the implementation and methods can be written in any order.
func build[T comparable](values ...T) *DoublyLinkedList[T] {
	l := &DoublyLinkedList[T]{}
	for _, v := range values {
		n := &node[T]{value: v}
		if l.head == nil {
			l.head = n
		} else {
			n.prev = l.tail
			l.tail.next = n
		}
		l.tail = n
		l.size++
	}
	return l
}

// cyclic returns a list whose last node's next points back at the node at index at,
// leaving prev deliberately inconsistent. walkForward and assertIntact must never be
// called on the result - tripping the maxNodes guard is exactly what they are for.
func cyclic[T comparable](at int, values ...T) *DoublyLinkedList[T] {
	l := build(values...)
	target := l.head
	for i := 0; i < at; i++ {
		target = target.next
	}
	l.tail.next = target
	return l
}

// assertIntact checks every structural invariant at once: the chain read forward from
// head, the same chain read backward from tail, the size counter, the nil terminators,
// and the mutual next/prev identity that values alone cannot prove. It reads l.size
// directly rather than calling Len, so it never borrows a method to judge another.
// Call it after every mutation, and after every failed mutation too - a rejected call
// must leave the list untouched.
func assertIntact[T comparable](t *testing.T, l *DoublyLinkedList[T], want []T) {
	t.Helper()

	if got := walkForward(l.head, maxNodes); !slices.Equal(got, want) {
		t.Errorf("chain from head = %v, want %v", got, want)
	}
	if got, w := walkBackward(l.tail, maxNodes), reversed(want); !slices.Equal(got, w) {
		t.Errorf("chain from tail = %v, want %v", got, w)
	}
	if l.size != len(want) {
		t.Errorf("size field = %d, want %d", l.size, len(want))
	}

	if len(want) == 0 {
		if l.head != nil {
			t.Errorf("head = node(%v), want nil on an empty list", l.head.value)
		}
		if l.tail != nil {
			t.Errorf("tail = node(%v), want nil on an empty list", l.tail.value)
		}
		return
	}

	if l.head == nil || l.tail == nil {
		t.Fatalf("head = %v, tail = %v, want both non-nil for %v", l.head, l.tail, want)
	}
	if l.head.prev != nil {
		t.Errorf("head.prev = node(%v), want nil", l.head.prev.value)
	}
	if l.tail.next != nil {
		t.Errorf("tail.next = node(%v), want nil", l.tail.next.value)
	}

	// Matching values in both directions is not proof: with duplicates a stale prev
	// still spells the right sequence. Compare the nodes themselves.
	for n := l.head; n.next != nil; n = n.next {
		if n.next.prev != n {
			t.Errorf("node(%v).next.prev is a different node, want node(%v) itself - the backward link is stale", n.value, n.value)
		}
	}
	for n := l.tail; n.prev != nil; n = n.prev {
		if n.prev.next != n {
			t.Errorf("node(%v).prev.next is a different node, want node(%v) itself - the forward link is stale", n.value, n.value)
		}
	}
}

// captureStdout redirects os.Stdout for the duration of f and returns what was written.
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

// TestDoublyLinked pins the shape of the structure rather than any one method. Head,
// tail and prev belong here, and the singly linked list's TestSinglyLinked asserts the
// exact opposite of this - the two tests together are what keep the structures from
// blurring into each other.
func TestDoublyLinked(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  reflect.Type
		want []string
	}{
		{"list", reflect.TypeOf(DoublyLinkedList[int]{}), []string{"head", "tail", "size"}},
		{"node", reflect.TypeOf(node[int]{}), []string{"value", "next", "prev"}},
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

// TestEmpty is the case most implementations get wrong, so every method gets a subtest
// against a freshly built empty list.
func TestEmpty(t *testing.T) {
	t.Run("Len", func(t *testing.T) {
		if got := build[int]().Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
	})
	t.Run("Get", func(t *testing.T) {
		l := build[int]()
		got, err := l.Get(0)
		if !errors.Is(err, ErrIndexOutOfRange) {
			t.Errorf("Get(0) error = %v, want ErrIndexOutOfRange", err)
		}
		if got != 0 {
			t.Errorf("Get(0) = %d, want the zero value", got)
		}
		assertIntact(t, l, nil)
	})
	t.Run("Insert at 0 succeeds", func(t *testing.T) {
		l := build[int]()
		if err := l.Insert(0, 10); err != nil {
			t.Fatalf("Insert(0, 10) error = %v, want nil", err)
		}
		assertIntact(t, l, []int{10})
	})
	t.Run("Insert past 0 fails", func(t *testing.T) {
		l := build[int]()
		if err := l.Insert(1, 10); !errors.Is(err, ErrIndexOutOfRange) {
			t.Errorf("Insert(1, 10) error = %v, want ErrIndexOutOfRange", err)
		}
		assertIntact(t, l, nil)
	})
	t.Run("PopFront", func(t *testing.T) {
		l := build[int]()
		got, err := l.PopFront()
		if !errors.Is(err, ErrEmptyList) {
			t.Errorf("PopFront() error = %v, want ErrEmptyList", err)
		}
		if got != 0 {
			t.Errorf("PopFront() = %d, want the zero value", got)
		}
		assertIntact(t, l, nil)
	})
	t.Run("Pop", func(t *testing.T) {
		l := build[int]()
		got, err := l.Pop()
		if !errors.Is(err, ErrEmptyList) {
			t.Errorf("Pop() error = %v, want ErrEmptyList", err)
		}
		if got != 0 {
			t.Errorf("Pop() = %d, want the zero value", got)
		}
		assertIntact(t, l, nil)
	})
	t.Run("RemoveAt", func(t *testing.T) {
		l := build[int]()
		got, err := l.RemoveAt(0)
		if !errors.Is(err, ErrIndexOutOfRange) {
			t.Errorf("RemoveAt(0) error = %v, want ErrIndexOutOfRange", err)
		}
		if got != 0 {
			t.Errorf("RemoveAt(0) = %d, want the zero value", got)
		}
		assertIntact(t, l, nil)
	})
	t.Run("Remove", func(t *testing.T) {
		l := build[int]()
		if got := l.Remove(10); got {
			t.Errorf("Remove(10) = true, want false")
		}
		assertIntact(t, l, nil)
	})
	t.Run("IndexOf", func(t *testing.T) {
		l := build[int]()
		got, ok := l.IndexOf(10)
		if ok {
			t.Errorf("IndexOf(10) ok = true, want false")
		}
		if got != 0 {
			t.Errorf("IndexOf(10) index = %d, want 0", got)
		}
	})
	t.Run("Contains", func(t *testing.T) {
		if got := build[int]().Contains(10); got {
			t.Errorf("Contains(10) = true, want false")
		}
	})
	t.Run("Reverse", func(t *testing.T) {
		l := build[int]()
		l.Reverse()
		assertIntact(t, l, nil)
	})
	t.Run("HasCycle", func(t *testing.T) {
		if got := build[int]().HasCycle(); got {
			t.Errorf("HasCycle() = true, want false")
		}
	})
	t.Run("ToSlice", func(t *testing.T) {
		if got := build[int]().ToSlice(); len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
		}
	})
	t.Run("ToSliceReverse", func(t *testing.T) {
		if got := build[int]().ToSliceReverse(); len(got) != 0 {
			t.Errorf("ToSliceReverse() = %v, want empty", got)
		}
	})
	t.Run("String", func(t *testing.T) {
		if got := build[int]().String(); got != "nil" {
			t.Errorf("String() = %q, want %q", got, "nil")
		}
	})
}

func TestNew(t *testing.T) {
	l := New[int]()
	if l == nil {
		t.Fatal("New() = nil, want an empty list")
	}
	assertIntact(t, l, nil)
}

func TestLen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
	}{
		{"empty", nil},
		{"one", []int{10}},
		{"two", []int{10, 20}},
		{"many", []int{10, 20, 30, 40}},
		{"duplicates", []int{10, 10, 10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := build(tc.start...).Len(); got != len(tc.start) {
				t.Errorf("Len() = %d, want %d", got, len(tc.start))
			}
		})
	}
}

func TestAppend(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		value int
		want  []int
	}{
		{"onto empty", nil, 10, []int{10}},
		{"onto one", []int{10}, 20, []int{10, 20}},
		{"onto two", []int{10, 20}, 30, []int{10, 20, 30}},
		{"duplicate value", []int{10, 20}, 10, []int{10, 20, 10}},
		{"zero value", []int{10}, 0, []int{10, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			l.Append(tc.value)
			assertIntact(t, l, tc.want)
		})
	}

	t.Run("repeated onto empty", func(t *testing.T) {
		l := build[int]()
		for i, v := range []int{10, 20, 30} {
			l.Append(v)
			assertIntact(t, l, []int{10, 20, 30}[:i+1])
		}
	})
}

func TestPrepend(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		value int
		want  []int
	}{
		{"onto empty", nil, 10, []int{10}},
		{"onto one", []int{20}, 10, []int{10, 20}},
		{"onto two", []int{20, 30}, 10, []int{10, 20, 30}},
		{"duplicate value", []int{10, 20}, 10, []int{10, 10, 20}},
		{"zero value", []int{10}, 0, []int{0, 10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			l.Prepend(tc.value)
			assertIntact(t, l, tc.want)
		})
	}

	t.Run("repeated onto empty", func(t *testing.T) {
		l := build[int]()
		for i, v := range []int{30, 20, 10} {
			l.Prepend(v)
			assertIntact(t, l, []int{10, 20, 30}[2-i:])
		}
	})
}

func TestGet(t *testing.T) {
	// Every index of lists of both parities, because Get walks from whichever end is
	// nearer and the midpoint is where the two walks meet.
	for _, start := range [][]int{
		{10},
		{10, 20},
		{10, 20, 30},
		{10, 20, 30, 40},
		{10, 20, 30, 40, 50},
	} {
		t.Run(fmt.Sprintf("len %d, every index", len(start)), func(t *testing.T) {
			l := build(start...)
			for i, want := range start {
				got, err := l.Get(i)
				if err != nil {
					t.Fatalf("Get(%d) error = %v, want nil", i, err)
				}
				if got != want {
					t.Errorf("Get(%d) = %v, want %v", i, got, want)
				}
			}
			assertIntact(t, l, start)
		})
	}

	for _, tc := range []struct {
		name  string
		index int
	}{
		{"negative", -1},
		{"far negative", -99},
		{"one past tail", 5},
		{"far past tail", 99},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := []int{10, 20, 30, 40, 50}
			l := build(start...)
			got, err := l.Get(tc.index)
			if !errors.Is(err, ErrIndexOutOfRange) {
				t.Errorf("Get(%d) error = %v, want ErrIndexOutOfRange", tc.index, err)
			}
			if got != 0 {
				t.Errorf("Get(%d) = %v, want the zero value on the error path", tc.index, got)
			}
			assertIntact(t, l, start)
		})
	}

	t.Run("zero value is retrievable", func(t *testing.T) {
		l := build(0, 0)
		got, err := l.Get(1)
		if err != nil {
			t.Fatalf("Get(1) error = %v, want nil", err)
		}
		if got != 0 {
			t.Errorf("Get(1) = %v, want 0", got)
		}
	})
}

func TestInsert(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		index int
		value int
		want  []int
	}{
		{"into empty", nil, 0, 10, []int{10}},
		{"at head", []int{20, 30}, 0, 10, []int{10, 20, 30}},
		{"in middle", []int{10, 30}, 1, 20, []int{10, 20, 30}},
		{"before tail", []int{10, 20, 40}, 2, 30, []int{10, 20, 30, 40}},
		{"at Len appends", []int{10, 20}, 2, 30, []int{10, 20, 30}},
		{"into one at head", []int{20}, 0, 10, []int{10, 20}},
		{"into one at Len", []int{10}, 1, 20, []int{10, 20}},
		{"duplicate value", []int{10, 20}, 1, 10, []int{10, 10, 20}},
		{"zero value", []int{10, 20}, 1, 0, []int{10, 0, 20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			if err := l.Insert(tc.index, tc.value); err != nil {
				t.Fatalf("Insert(%d, %d) error = %v, want nil", tc.index, tc.value, err)
			}
			assertIntact(t, l, tc.want)
		})
	}

	for _, tc := range []struct {
		name  string
		index int
	}{
		{"negative", -1},
		{"past Len", 4},
		{"far past Len", 99},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := []int{10, 20, 30}
			l := build(start...)
			if err := l.Insert(tc.index, 99); !errors.Is(err, ErrIndexOutOfRange) {
				t.Errorf("Insert(%d, 99) error = %v, want ErrIndexOutOfRange", tc.index, err)
			}
			assertIntact(t, l, start)
		})
	}
}

func TestPopFront(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
		rest  []int
	}{
		{"one", []int{10}, 10, nil},
		{"two", []int{10, 20}, 10, []int{20}},
		{"many", []int{10, 20, 30}, 10, []int{20, 30}},
		{"duplicates", []int{10, 10}, 10, []int{10}},
		{"zero value", []int{0, 10}, 0, []int{10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			got, err := l.PopFront()
			if err != nil {
				t.Fatalf("PopFront() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("PopFront() = %v, want %v", got, tc.want)
			}
			assertIntact(t, l, tc.rest)
		})
	}

	t.Run("drains in order", func(t *testing.T) {
		start := []int{10, 20, 30}
		l := build(start...)
		for i, want := range start {
			got, err := l.PopFront()
			if err != nil {
				t.Fatalf("PopFront() #%d error = %v, want nil", i, err)
			}
			if got != want {
				t.Errorf("PopFront() #%d = %v, want %v", i, got, want)
			}
			assertIntact(t, l, start[i+1:])
		}
		if _, err := l.PopFront(); !errors.Is(err, ErrEmptyList) {
			t.Errorf("PopFront() on the drained list error = %v, want ErrEmptyList", err)
		}
	})
}

func TestPop(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
		rest  []int
	}{
		{"one", []int{10}, 10, nil},
		{"two", []int{10, 20}, 20, []int{10}},
		{"many", []int{10, 20, 30}, 30, []int{10, 20}},
		{"duplicates", []int{10, 10}, 10, []int{10}},
		{"zero value", []int{10, 0}, 0, []int{10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			got, err := l.Pop()
			if err != nil {
				t.Fatalf("Pop() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Pop() = %v, want %v", got, tc.want)
			}
			assertIntact(t, l, tc.rest)
		})
	}

	t.Run("drains in reverse order", func(t *testing.T) {
		start := []int{10, 20, 30}
		l := build(start...)
		for i := len(start) - 1; i >= 0; i-- {
			got, err := l.Pop()
			if err != nil {
				t.Fatalf("Pop() at %d error = %v, want nil", i, err)
			}
			if got != start[i] {
				t.Errorf("Pop() at %d = %v, want %v", i, got, start[i])
			}
			assertIntact(t, l, start[:i])
		}
		if _, err := l.Pop(); !errors.Is(err, ErrEmptyList) {
			t.Errorf("Pop() on the drained list error = %v, want ErrEmptyList", err)
		}
	})
}

func TestRemoveAt(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		index int
		want  int
		rest  []int
	}{
		{"only node", []int{10}, 0, 10, nil},
		{"head", []int{10, 20, 30}, 0, 10, []int{20, 30}},
		{"middle", []int{10, 20, 30}, 1, 20, []int{10, 30}},
		{"tail", []int{10, 20, 30}, 2, 30, []int{10, 20}},
		{"head of two", []int{10, 20}, 0, 10, []int{20}},
		{"tail of two", []int{10, 20}, 1, 20, []int{10}},
		{"duplicate removes first", []int{10, 10, 20}, 0, 10, []int{10, 20}},
		{"zero value", []int{10, 0, 20}, 1, 0, []int{10, 20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			got, err := l.RemoveAt(tc.index)
			if err != nil {
				t.Fatalf("RemoveAt(%d) error = %v, want nil", tc.index, err)
			}
			if got != tc.want {
				t.Errorf("RemoveAt(%d) = %v, want %v", tc.index, got, tc.want)
			}
			assertIntact(t, l, tc.rest)
		})
	}

	for _, tc := range []struct {
		name  string
		index int
	}{
		{"negative", -1},
		{"at Len", 3},
		{"far past tail", 99},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := []int{10, 20, 30}
			l := build(start...)
			got, err := l.RemoveAt(tc.index)
			if !errors.Is(err, ErrIndexOutOfRange) {
				t.Errorf("RemoveAt(%d) error = %v, want ErrIndexOutOfRange", tc.index, err)
			}
			if got != 0 {
				t.Errorf("RemoveAt(%d) = %v, want the zero value on the error path", tc.index, got)
			}
			assertIntact(t, l, start)
		})
	}
}

func TestRemove(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		value int
		want  bool
		rest  []int
	}{
		{"only node", []int{10}, 10, true, nil},
		{"head", []int{10, 20, 30}, 10, true, []int{20, 30}},
		{"middle", []int{10, 20, 30}, 20, true, []int{10, 30}},
		{"tail", []int{10, 20, 30}, 30, true, []int{10, 20}},
		{"first of duplicates only", []int{10, 20, 10}, 10, true, []int{20, 10}},
		{"adjacent duplicates", []int{10, 10, 10}, 10, true, []int{10, 10}},
		{"zero value", []int{10, 0, 20}, 0, true, []int{10, 20}},
		{"absent", []int{10, 20}, 99, false, []int{10, 20}},
		{"absent from one", []int{10}, 99, false, []int{10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			if got := l.Remove(tc.value); got != tc.want {
				t.Errorf("Remove(%v) = %v, want %v", tc.value, got, tc.want)
			}
			assertIntact(t, l, tc.rest)
		})
	}
}

func TestIndexOf(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		value int
		want  int
		ok    bool
	}{
		{"head", []int{10, 20, 30}, 10, 0, true},
		{"middle", []int{10, 20, 30}, 20, 1, true},
		{"tail", []int{10, 20, 30}, 30, 2, true},
		{"only node", []int{10}, 10, 0, true},
		{"first of duplicates", []int{10, 20, 10}, 10, 0, true},
		{"zero value present", []int{10, 0}, 0, 1, true},
		{"absent", []int{10, 20}, 99, 0, false},
		{"zero value absent", []int{10, 20}, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			got, ok := l.IndexOf(tc.value)
			if ok != tc.ok {
				t.Errorf("IndexOf(%v) ok = %v, want %v", tc.value, ok, tc.ok)
			}
			if got != tc.want {
				t.Errorf("IndexOf(%v) index = %d, want %d", tc.value, got, tc.want)
			}
			assertIntact(t, l, tc.start)
		})
	}
}

func TestContains(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		value int
		want  bool
	}{
		{"head", []int{10, 20, 30}, 10, true},
		{"middle", []int{10, 20, 30}, 20, true},
		{"tail", []int{10, 20, 30}, 30, true},
		{"only node", []int{10}, 10, true},
		{"zero value present", []int{10, 0}, 0, true},
		{"absent", []int{10, 20}, 99, false},
		{"zero value absent", []int{10, 20}, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			if got := l.Contains(tc.value); got != tc.want {
				t.Errorf("Contains(%v) = %v, want %v", tc.value, got, tc.want)
			}
			assertIntact(t, l, tc.start)
		})
	}
}

func TestReverse(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  []int
	}{
		{"empty", nil, nil},
		{"one", []int{10}, []int{10}},
		{"two", []int{10, 20}, []int{20, 10}},
		{"many", []int{10, 20, 30, 40}, []int{40, 30, 20, 10}},
		{"duplicates", []int{10, 10, 20}, []int{20, 10, 10}},
		{"zero values", []int{0, 10, 0}, []int{0, 10, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			l.Reverse()
			assertIntact(t, l, tc.want)
		})
	}

	// assertIntact already proves head and tail swapped and every prev was rebuilt.
	// The round trip adds the guarantee that no node was dropped or duplicated along
	// the way, which a single flip of a short list can hide.
	t.Run("twice is identity", func(t *testing.T) {
		start := []int{10, 20, 30}
		l := build(start...)
		l.Reverse()
		l.Reverse()
		assertIntact(t, l, start)
	})
}

func TestHasCycle(t *testing.T) {
	t.Run("acyclic", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			start []int
		}{
			{"empty", nil},
			{"one", []int{10}},
			{"two", []int{10, 20}},
			{"many", []int{10, 20, 30, 40, 50}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				if got := build(tc.start...).HasCycle(); got {
					t.Errorf("HasCycle() = true, want false")
				}
			})
		}
	})

	t.Run("cyclic", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			at    int
			start []int
		}{
			{"self loop", 0, []int{10}},
			{"tail to head", 0, []int{10, 20, 30}},
			{"tail to middle", 1, []int{10, 20, 30, 40}},
			{"tail to itself", 3, []int{10, 20, 30, 40}},
			{"odd length loop", 1, []int{10, 20, 30}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				if got := cyclic(tc.at, tc.start...).HasCycle(); !got {
					t.Errorf("HasCycle() = false, want true")
				}
			})
		}
	})
}

func TestToSlice(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
	}{
		{"one", []int{10}},
		{"two", []int{10, 20}},
		{"many", []int{10, 20, 30, 40}},
		{"duplicates", []int{10, 10, 10}},
		{"zero values", []int{0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			if got := l.ToSlice(); !slices.Equal(got, tc.start) {
				t.Errorf("ToSlice() = %v, want %v", got, tc.start)
			}
			assertIntact(t, l, tc.start)
		})
	}

	t.Run("empty is non-nil", func(t *testing.T) {
		got := build[int]().ToSlice()
		if got == nil {
			t.Fatal("ToSlice() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
		}
	})

	t.Run("result is a copy", func(t *testing.T) {
		start := []int{10, 20, 30}
		l := build(start...)
		got := l.ToSlice()
		got[0] = 99
		assertIntact(t, l, start)
	})
}

func TestToSliceReverse(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  []int
	}{
		{"one", []int{10}, []int{10}},
		{"two", []int{10, 20}, []int{20, 10}},
		{"many", []int{10, 20, 30, 40}, []int{40, 30, 20, 10}},
		{"duplicates", []int{10, 20, 10}, []int{10, 20, 10}},
		{"zero values", []int{0, 10}, []int{10, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			if got := l.ToSliceReverse(); !slices.Equal(got, tc.want) {
				t.Errorf("ToSliceReverse() = %v, want %v", got, tc.want)
			}
			assertIntact(t, l, tc.start)
		})
	}

	t.Run("empty is non-nil", func(t *testing.T) {
		got := build[int]().ToSliceReverse()
		if got == nil {
			t.Fatal("ToSliceReverse() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSliceReverse() = %v, want empty", got)
		}
	})

	t.Run("result is a copy", func(t *testing.T) {
		start := []int{10, 20, 30}
		l := build(start...)
		got := l.ToSliceReverse()
		got[0] = 99
		assertIntact(t, l, start)
	})
}

func TestString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "nil"},
		{"one", []int{10}, "10 <-> nil"},
		{"two", []int{10, 20}, "10 <-> 20 <-> nil"},
		{"many", []int{10, 20, 30}, "10 <-> 20 <-> 30 <-> nil"},
		{"zero value", []int{0}, "0 <-> nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			if got := l.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
			assertIntact(t, l, tc.start)
		})
	}
}

// TestPrintList delegates to String by design - printing the rendered chain is the
// behavior under test - so it goes red if String is broken.
func TestPrintList(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "nil\n"},
		{"one", []int{10}, "10 <-> nil\n"},
		{"many", []int{10, 20, 30}, "10 <-> 20 <-> 30 <-> nil\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			got := captureStdout(t, l.PrintList)
			if got != tc.want {
				t.Errorf("PrintList() wrote %q, want %q", got, tc.want)
			}
			assertIntact(t, l, tc.start)
		})
	}
}

func ExampleDoublyLinkedList() {
	l := New[int]()
	l.Append(10)
	l.Append(20)
	l.Prepend(5)
	l.PrintList()
	// Output: 5 <-> 10 <-> 20 <-> nil
}
