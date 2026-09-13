// White-box tests for the singly linked list.
//
//	go test ./singlylinkedlist/ -v              # the whole suite
//	go test ./singlylinkedlist/ -run TestPop -v # one method
//
// The list stores head and size, and the node stores value and next. There is no tail
// pointer and no prev pointer: the second pointer in either place is what makes a list
// doubly linked, so both belong to that structure and not this one. The word "tail" in
// this file names a position - the last node - never a stored field. TestSinglyLinked
// enforces that, so the distinction cannot erode into a half-doubly-linked list.
//
// The file is package singlylinkedlist, not ..._test, so the oracle helpers below
// can read head, next and size while those fields stay unexported. Every
// assertion goes through those helpers: no method is ever used to verify another,
// so a red test names exactly one unwritten method.
//
// Three deliberate exceptions, each noted again above the test itself:
//   - TestSinglyLinked guards the shape of the structure rather than a method, so it
//     is the one test here that passes against the stubs.
//   - TestPrintList delegates to String, because printing the rendered chain is the
//     behavior under test.
//   - ExampleSinglyLinkedList exercises New, Append, Prepend and PrintList together,
//     because a runnable example is by definition an integration.
package singlylinkedlist

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"slices"
	"testing"
)

// maxNodes bounds every walking helper, so a chain wired into a cycle fails the
// suite instead of hanging it.
const maxNodes = 1000

// walk collects values by following next from head, independent of any method.
func walk[T comparable](head *node[T], limit int) []T {
	var values []T
	for n := head; n != nil; n = n.next {
		if len(values) > limit {
			panic("walked past the limit - the chain has a cycle")
		}
		values = append(values, n.value)
	}
	return values
}

// build wires nodes by hand rather than calling Append, so every test in this file
// is independent of the implementation and methods can be written in any order. The
// list keeps no tail pointer, so build tracks the last node in a local.
func build[T comparable](values ...T) *SinglyLinkedList[T] {
	l := &SinglyLinkedList[T]{}
	var last *node[T]
	for _, v := range values {
		n := &node[T]{value: v}
		if last == nil {
			l.head = n
		} else {
			last.next = n
		}
		last = n
		l.size++
	}
	return l
}

// cyclic returns a list whose last node points back at the node at index at.
// walk and assertIntact must never be called on the result - tripping the maxNodes
// guard is exactly what they are built to do.
func cyclic[T comparable](at int, values ...T) *SinglyLinkedList[T] {
	l := build(values...)
	target := l.head
	for i := 0; i < at; i++ {
		target = target.next
	}
	last := l.head
	for last.next != nil {
		last = last.next
	}
	last.next = target
	return l
}

// assertIntact checks every structural invariant at once: the chain from head and the
// size counter, which with no tail pointer is the whole of the list's state. walk
// stops at the first nil next, so a chain that fails to terminate trips the maxNodes
// guard rather than passing. It reads l.size directly rather than calling Len, so it
// stays usable before Len is written. Call it after every mutation, and after every
// failed mutation too - a rejected call must leave the list untouched.
func assertIntact[T comparable](t *testing.T, l *SinglyLinkedList[T], want []T) {
	t.Helper()

	if got := walk(l.head, maxNodes); !slices.Equal(got, want) {
		t.Errorf("chain from head = %v, want %v", got, want)
	}
	if l.size != len(want) {
		t.Errorf("size field = %d, want %d", l.size, len(want))
	}
	if len(want) == 0 && l.head != nil {
		t.Errorf("head = node(%v), want nil on an empty list", l.head.value)
	}
}

// captureStdout redirects os.Stdout for the duration of f and returns what was
// written to it.
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

// TestSinglyLinked guards the shape of the structure rather than any one method, so it
// is the only test here that passes against the stubs. A singly linked list stores head
// and size, and each node stores value and next. A tail pointer on the list, or a prev
// pointer on the node, is the doubly linked list's design: either one would buy back an
// O(1) Append or an O(1) Pop by quietly turning this into a different structure, and
// every cost documented on the stubs would stop being true.
func TestSinglyLinked(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  reflect.Type
		want []string
	}{
		{"list", reflect.TypeOf(SinglyLinkedList[int]{}), []string{"head", "size"}},
		{"node", reflect.TypeOf(node[int]{}), []string{"value", "next"}},
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

// TestEmpty is the case most implementations get wrong, so every method gets a
// subtest against a freshly built empty list.
func TestEmpty(t *testing.T) {
	t.Run("Len", func(t *testing.T) {
		if got := build[int]().Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
	})
	t.Run("Append", func(t *testing.T) {
		l := build[int]()
		l.Append(7)
		assertIntact(t, l, []int{7})
	})
	t.Run("Prepend", func(t *testing.T) {
		l := build[int]()
		l.Prepend(7)
		assertIntact(t, l, []int{7})
	})
	t.Run("Get", func(t *testing.T) {
		l := build[int]()
		if _, err := l.Get(0); !errors.Is(err, ErrIndexOutOfRange) {
			t.Errorf("Get(0) error = %v, want ErrIndexOutOfRange", err)
		}
		assertIntact(t, l, nil)
	})
	t.Run("Insert", func(t *testing.T) {
		l := build[int]()
		if err := l.Insert(0, 7); err != nil {
			t.Errorf("Insert(0, 7) error = %v, want nil (index 0 == Len is the append slot)", err)
		}
		assertIntact(t, l, []int{7})
	})
	t.Run("PopFront", func(t *testing.T) {
		l := build[int]()
		if _, err := l.PopFront(); !errors.Is(err, ErrEmptyList) {
			t.Errorf("PopFront() error = %v, want ErrEmptyList", err)
		}
		assertIntact(t, l, nil)
	})
	t.Run("Pop", func(t *testing.T) {
		l := build[int]()
		if _, err := l.Pop(); !errors.Is(err, ErrEmptyList) {
			t.Errorf("Pop() error = %v, want ErrEmptyList", err)
		}
		assertIntact(t, l, nil)
	})
	t.Run("RemoveAt", func(t *testing.T) {
		l := build[int]()
		if _, err := l.RemoveAt(0); !errors.Is(err, ErrIndexOutOfRange) {
			t.Errorf("RemoveAt(0) error = %v, want ErrIndexOutOfRange", err)
		}
		assertIntact(t, l, nil)
	})
	t.Run("Remove", func(t *testing.T) {
		l := build[int]()
		if got := l.Remove(7); got {
			t.Errorf("Remove(7) = true, want false on an empty list")
		}
		assertIntact(t, l, nil)
	})
	t.Run("IndexOf", func(t *testing.T) {
		l := build[int]()
		if _, ok := l.IndexOf(7); ok {
			t.Errorf("IndexOf(7) ok = true, want false on an empty list")
		}
	})
	t.Run("Contains", func(t *testing.T) {
		if got := build[int]().Contains(7); got {
			t.Errorf("Contains(7) = true, want false on an empty list")
		}
	})
	t.Run("Reverse", func(t *testing.T) {
		l := build[int]()
		l.Reverse()
		assertIntact(t, l, nil)
	})
	t.Run("HasCycle", func(t *testing.T) {
		if got := build[int]().HasCycle(); got {
			t.Errorf("HasCycle() = true, want false on an empty list")
		}
	})
	t.Run("ToSlice", func(t *testing.T) {
		got := build[int]().ToSlice()
		if got == nil {
			t.Error("ToSlice() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
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
		t.Fatal("New() = nil, want a usable empty list")
	}
	assertIntact(t, l, nil)
}

func TestLen(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values []int
	}{
		{"empty", nil},
		{"one", []int{10}},
		{"two", []int{10, 20}},
		{"many", []int{10, 20, 30, 40, 50}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := build(tc.values...).Len(); got != len(tc.values) {
				t.Errorf("Len() = %d, want %d", got, len(tc.values))
			}
		})
	}
}

func TestAppend(t *testing.T) {
	t.Run("onto empty", func(t *testing.T) {
		l := build[int]()
		l.Append(10)
		assertIntact(t, l, []int{10})
	})
	t.Run("onto one", func(t *testing.T) {
		l := build(10)
		l.Append(20)
		assertIntact(t, l, []int{10, 20})
	})
	t.Run("onto many", func(t *testing.T) {
		l := build(10, 20, 30)
		l.Append(40)
		assertIntact(t, l, []int{10, 20, 30, 40})
	})
	t.Run("duplicates are kept", func(t *testing.T) {
		l := build(10)
		l.Append(10)
		assertIntact(t, l, []int{10, 10})
	})
	t.Run("zero value", func(t *testing.T) {
		l := build[int]()
		l.Append(0)
		assertIntact(t, l, []int{0})
	})
}

func TestPrepend(t *testing.T) {
	t.Run("onto empty", func(t *testing.T) {
		l := build[int]()
		l.Prepend(10)
		assertIntact(t, l, []int{10})
	})
	t.Run("onto one", func(t *testing.T) {
		l := build(20)
		l.Prepend(10)
		assertIntact(t, l, []int{10, 20})
	})
	t.Run("onto many", func(t *testing.T) {
		l := build(20, 30, 40)
		l.Prepend(10)
		assertIntact(t, l, []int{10, 20, 30, 40})
	})
	t.Run("duplicates are kept", func(t *testing.T) {
		l := build(10)
		l.Prepend(10)
		assertIntact(t, l, []int{10, 10})
	})
}

func TestGet(t *testing.T) {
	values := []int{10, 20, 30, 40, 50}

	for _, tc := range []struct {
		name  string
		index int
		want  int
	}{
		{"head", 0, 10},
		{"second", 1, 20},
		{"middle", 2, 30},
		{"tail", 4, 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(values...)
			got, err := l.Get(tc.index)
			if err != nil {
				t.Fatalf("Get(%d) error = %v, want nil", tc.index, err)
			}
			if got != tc.want {
				t.Errorf("Get(%d) = %d, want %d", tc.index, got, tc.want)
			}
			assertIntact(t, l, values)
		})
	}

	for _, tc := range []struct {
		name  string
		index int
	}{
		{"negative", -1},
		{"one past tail", 5},
		{"far past tail", 99},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(values...)
			_, err := l.Get(tc.index)
			if !errors.Is(err, ErrIndexOutOfRange) {
				t.Errorf("Get(%d) error = %v, want ErrIndexOutOfRange", tc.index, err)
			}
			assertIntact(t, l, values)
		})
	}

	// A zero T must be reachable, and distinguishable from "absent" by the error.
	t.Run("zero value is a real element", func(t *testing.T) {
		l := build(0, 0)
		got, err := l.Get(1)
		if err != nil {
			t.Fatalf("Get(1) error = %v, want nil", err)
		}
		if got != 0 {
			t.Errorf("Get(1) = %d, want 0", got)
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
		left  []int
	}{
		{"one element", []int{10}, 10, nil},
		{"two elements", []int{10, 20}, 10, []int{20}},
		{"many", []int{10, 20, 30}, 10, []int{20, 30}},
		{"zero value", []int{0, 20}, 0, []int{20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			got, err := l.PopFront()
			if err != nil {
				t.Fatalf("PopFront() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("PopFront() = %d, want %d", got, tc.want)
			}
			assertIntact(t, l, tc.left)
		})
	}

	t.Run("empty reports the sentinel", func(t *testing.T) {
		l := build[int]()
		_, err := l.PopFront()
		if !errors.Is(err, ErrEmptyList) {
			t.Errorf("PopFront() error = %v, want ErrEmptyList", err)
		}
		assertIntact(t, l, nil)
	})

	t.Run("drain then refill", func(t *testing.T) {
		l := build(10, 20)
		for i := 0; i < 2; i++ {
			if _, err := l.PopFront(); err != nil {
				t.Fatalf("PopFront() error = %v, want nil", err)
			}
		}
		assertIntact(t, l, nil)
		l.Prepend(30)
		assertIntact(t, l, []int{30})
	})
}

func TestPop(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
		left  []int
	}{
		{"one element", []int{10}, 10, nil},
		{"two elements", []int{10, 20}, 20, []int{10}},
		{"many", []int{10, 20, 30}, 30, []int{10, 20}},
		{"zero value", []int{10, 0}, 0, []int{10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			got, err := l.Pop()
			if err != nil {
				t.Fatalf("Pop() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Pop() = %d, want %d", got, tc.want)
			}
			assertIntact(t, l, tc.left)
		})
	}

	t.Run("empty reports the sentinel", func(t *testing.T) {
		l := build[int]()
		_, err := l.Pop()
		if !errors.Is(err, ErrEmptyList) {
			t.Errorf("Pop() error = %v, want ErrEmptyList", err)
		}
		assertIntact(t, l, nil)
	})

	t.Run("drain then refill", func(t *testing.T) {
		l := build(10, 20)
		for i := 0; i < 2; i++ {
			if _, err := l.Pop(); err != nil {
				t.Fatalf("Pop() error = %v, want nil", err)
			}
		}
		assertIntact(t, l, nil)
		l.Append(30)
		assertIntact(t, l, []int{30})
	})
}

func TestRemoveAt(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		index int
		want  int
		left  []int
	}{
		{"only node", []int{10}, 0, 10, nil},
		{"head", []int{10, 20, 30}, 0, 10, []int{20, 30}},
		{"middle", []int{10, 20, 30}, 1, 20, []int{10, 30}},
		{"tail", []int{10, 20, 30}, 2, 30, []int{10, 20}},
		{"head of two", []int{10, 20}, 0, 10, []int{20}},
		{"tail of two", []int{10, 20}, 1, 20, []int{10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			got, err := l.RemoveAt(tc.index)
			if err != nil {
				t.Fatalf("RemoveAt(%d) error = %v, want nil", tc.index, err)
			}
			if got != tc.want {
				t.Errorf("RemoveAt(%d) = %d, want %d", tc.index, got, tc.want)
			}
			assertIntact(t, l, tc.left)
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
			if _, err := l.RemoveAt(tc.index); !errors.Is(err, ErrIndexOutOfRange) {
				t.Errorf("RemoveAt(%d) error = %v, want ErrIndexOutOfRange", tc.index, err)
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
		left  []int
	}{
		{"only node", []int{10}, 10, true, nil},
		{"head", []int{10, 20, 30}, 10, true, []int{20, 30}},
		{"middle", []int{10, 20, 30}, 20, true, []int{10, 30}},
		{"tail", []int{10, 20, 30}, 30, true, []int{10, 20}},
		{"first duplicate only", []int{10, 20, 10}, 10, true, []int{20, 10}},
		{"absent leaves list alone", []int{10, 20}, 99, false, []int{10, 20}},
		{"zero value is removable", []int{10, 0, 20}, 0, true, []int{10, 20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			if got := l.Remove(tc.value); got != tc.want {
				t.Errorf("Remove(%d) = %v, want %v", tc.value, got, tc.want)
			}
			assertIntact(t, l, tc.left)
		})
	}
}

func TestIndexOf(t *testing.T) {
	for _, tc := range []struct {
		name      string
		start     []int
		value     int
		wantIndex int
		wantOK    bool
	}{
		{"head", []int{10, 20, 30}, 10, 0, true},
		{"middle", []int{10, 20, 30}, 20, 1, true},
		{"tail", []int{10, 20, 30}, 30, 2, true},
		{"first of duplicates", []int{10, 20, 10}, 10, 0, true},
		{"absent", []int{10, 20}, 99, 0, false},
		{"zero value present", []int{10, 0}, 0, 1, true},
		{"zero value absent", []int{10, 20}, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			gotIndex, gotOK := l.IndexOf(tc.value)
			if gotOK != tc.wantOK {
				t.Errorf("IndexOf(%d) ok = %v, want %v", tc.value, gotOK, tc.wantOK)
			}
			if gotOK && gotIndex != tc.wantIndex {
				t.Errorf("IndexOf(%d) index = %d, want %d", tc.value, gotIndex, tc.wantIndex)
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
		{"absent", []int{10, 20}, 99, false},
		{"zero value present", []int{10, 0}, 0, true},
		{"zero value absent", []int{10, 20}, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			if got := l.Contains(tc.value); got != tc.want {
				t.Errorf("Contains(%d) = %v, want %v", tc.value, got, tc.want)
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := build(tc.start...)
			l.Reverse()
			assertIntact(t, l, tc.want)
		})
	}

	// The round trip catches a Reverse that drops or duplicates a link: a chain that
	// survives being flipped twice has kept every node, in order, still terminating.
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

func TestString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "nil"},
		{"one", []int{10}, "10 -> nil"},
		{"two", []int{10, 20}, "10 -> 20 -> nil"},
		{"many", []int{10, 20, 30}, "10 -> 20 -> 30 -> nil"},
		{"zero value", []int{0}, "0 -> nil"},
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
// behavior under test - so it stays red until String is written.
func TestPrintList(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "nil\n"},
		{"one", []int{10}, "10 -> nil\n"},
		{"many", []int{10, 20, 30}, "10 -> 20 -> 30 -> nil\n"},
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

func ExampleSinglyLinkedList() {
	l := New[int]()
	l.Append(10)
	l.Append(20)
	l.Prepend(5)
	l.PrintList()
	// Output: 5 -> 10 -> 20 -> nil
}
