// White-box tests for the slice-backed stack.
//
//	go test ./stack/ -v                      # every stack in the package
//	go test ./stack/ -run TestStackPop -v    # one method
//
// Three stacks share this package and Go test function names are package-scoped, so
// every test here is prefixed with the type it covers: TestStackPop is this file's,
// TestLinkedStackPop and TestMinStackPop belong to their own. captureStdout is
// declared here and used by all three files, since one declaration per package is all
// Go allows.
//
// The oracle is the backing slice itself. assertStack compares s.items against the
// values bottom to top, so no method is ever used to verify another and a red test
// names exactly one broken method. That is also why the file is package stack rather
// than package stack_test.
//
// assertReleased is the Go-specific one. Pop shrinks the length, which leaves the
// popped element above the length and still inside the capacity, where the backing
// array keeps it alive for as long as the stack lives. For a stack of ints that is
// invisible; for a stack of pointers it is an unbounded leak. The type comment makes
// zeroing the vacated slot part of Pop's contract, so the suite checks it.
//
// Two deliberate exceptions, each noted again above the test itself:
//   - TestStackShape guards the shape of the structure rather than a method.
//   - ExampleStack exercises New, Push, PrintStack, Pop and Len together, because a
//     runnable example is by definition an integration.
package stack

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

// buildStack returns a stack holding values bottom to top, filling the backing slice
// directly rather than calling Push, so every test in this file is independent of the
// implementation and the methods can be written in any order.
func buildStack[T comparable](values ...T) *Stack[T] {
	return &Stack[T]{items: append([]T(nil), values...)}
}

// assertStack checks the whole structure at once: the elements bottom to top, which
// for this type is simply the backing slice. Call it after every mutation, and after
// every rejected one too - a call that returns an error must leave the stack
// untouched.
func assertStack[T comparable](t *testing.T, s *Stack[T], want []T) {
	t.Helper()
	if got := s.items; !slices.Equal(got, want) {
		t.Errorf("items bottom to top = %v, want %v", got, want)
	}
}

// assertReleased checks the cell just above the top, which Pop must have zeroed so
// the popped element is not kept alive by the backing array. It reads past the
// length and inside the capacity, which is exactly where a leaked element hides.
func assertReleased[T comparable](t *testing.T, s *Stack[T]) {
	t.Helper()
	n := len(s.items)
	if cap(s.items) <= n {
		return // nothing above the top to inspect
	}
	var zero T
	if above := s.items[:n+1][n]; above != zero {
		t.Errorf("the cell above the top holds %v, want the zero value - Pop left the popped element reachable through the backing array", above)
	}
}

// captureStdout redirects os.Stdout for the duration of f and returns what was
// written. Shared by all three stack suites in this package.
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

// TestStackShape pins the shape of the structure rather than any one method. One
// slice and nothing else: no separate size counter to drift out of step with the
// elements, which is half the reason this version is simpler than the linked one.
func TestStackShape(t *testing.T) {
	var got []string
	for _, f := range reflect.VisibleFields(reflect.TypeOf(Stack[int]{})) {
		got = append(got, f.Name)
	}
	if want := []string{"items"}; !slices.Equal(got, want) {
		t.Errorf("Stack fields = %v, want %v", got, want)
	}
}

// TestStackEmpty is the case most implementations get wrong, so every method gets a
// subtest against a freshly built empty stack.
func TestStackEmpty(t *testing.T) {
	t.Run("Len", func(t *testing.T) {
		if got := buildStack[int]().Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
	})
	t.Run("IsEmpty", func(t *testing.T) {
		if got := buildStack[int]().IsEmpty(); !got {
			t.Errorf("IsEmpty() = false, want true")
		}
	})
	t.Run("Push", func(t *testing.T) {
		s := buildStack[int]()
		s.Push(10)
		assertStack(t, s, []int{10})
	})
	t.Run("Pop", func(t *testing.T) {
		s := buildStack[int]()
		got, err := s.Pop()
		if !errors.Is(err, ErrEmptyStack) {
			t.Errorf("Pop() error = %v, want ErrEmptyStack", err)
		}
		if got != 0 {
			t.Errorf("Pop() = %d, want the zero value", got)
		}
		assertStack(t, s, nil)
	})
	t.Run("Peek", func(t *testing.T) {
		s := buildStack[int]()
		got, err := s.Peek()
		if !errors.Is(err, ErrEmptyStack) {
			t.Errorf("Peek() error = %v, want ErrEmptyStack", err)
		}
		if got != 0 {
			t.Errorf("Peek() = %d, want the zero value", got)
		}
		assertStack(t, s, nil)
	})
	t.Run("ToSlice", func(t *testing.T) {
		got := buildStack[int]().ToSlice()
		if got == nil {
			t.Fatal("ToSlice() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
		}
	})
	t.Run("String", func(t *testing.T) {
		if got := buildStack[int]().String(); got != "nil" {
			t.Errorf("String() = %q, want %q", got, "nil")
		}
	})
}

func TestStackNew(t *testing.T) {
	s := New[int]()
	if s == nil {
		t.Fatal("New() = nil, want an empty stack")
	}
	assertStack(t, s, nil)
}

func TestStackLen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
	}{
		{"empty", nil},
		{"one", []int{10}},
		{"two", []int{10, 20}},
		{"many", []int{10, 20, 30, 40}},
		{"duplicates", []int{10, 10, 10}},
		{"zero values", []int{0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildStack(tc.start...).Len(); got != len(tc.start) {
				t.Errorf("Len() = %d, want %d", got, len(tc.start))
			}
		})
	}
}

func TestStackIsEmpty(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  bool
	}{
		{"empty", nil, true},
		{"one", []int{10}, false},
		{"many", []int{10, 20, 30}, false},
		{"one zero value", []int{0}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildStack(tc.start...).IsEmpty(); got != tc.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStackPush(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		value int
		want  []int
	}{
		{"onto empty", nil, 10, []int{10}},
		{"onto one", []int{10}, 20, []int{10, 20}},
		{"onto many", []int{10, 20}, 30, []int{10, 20, 30}},
		{"duplicate value", []int{10, 20}, 10, []int{10, 20, 10}},
		{"zero value", []int{10}, 0, []int{10, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := buildStack(tc.start...)
			s.Push(tc.value)
			assertStack(t, s, tc.want)
		})
	}

	t.Run("repeated onto empty", func(t *testing.T) {
		s := buildStack[int]()
		want := []int{10, 20, 30}
		for i, v := range want {
			s.Push(v)
			assertStack(t, s, want[:i+1])
		}
	})

	// The reallocation Push is amortized over. A version that writes through a stale
	// slice header instead of reassigning it survives the small cases above and dies
	// here, at the first grow.
	t.Run("grows past the initial capacity", func(t *testing.T) {
		s := buildStack[int]()
		want := make([]int, 0, 100)
		for i := 0; i < 100; i++ {
			s.Push(i)
			want = append(want, i)
		}
		assertStack(t, s, want)
	})
}

func TestStackPop(t *testing.T) {
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
		{"zero value on top", []int{10, 0}, 0, []int{10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := buildStack(tc.start...)
			got, err := s.Pop()
			if err != nil {
				t.Fatalf("Pop() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Pop() = %v, want %v", got, tc.want)
			}
			assertStack(t, s, tc.rest)
			assertReleased(t, s)
		})
	}

	t.Run("drains last in first out", func(t *testing.T) {
		start := []int{10, 20, 30}
		s := buildStack(start...)
		for i := len(start) - 1; i >= 0; i-- {
			got, err := s.Pop()
			if err != nil {
				t.Fatalf("Pop() at %d error = %v, want nil", i, err)
			}
			if got != start[i] {
				t.Errorf("Pop() at %d = %v, want %v", i, got, start[i])
			}
			assertStack(t, s, start[:i])
			assertReleased(t, s)
		}
		if _, err := s.Pop(); !errors.Is(err, ErrEmptyStack) {
			t.Errorf("Pop() on the drained stack error = %v, want ErrEmptyStack", err)
		}
	})

	// The leak made visible: with pointers, a slot Pop failed to zero is a live
	// reference to an object the caller believes it handed back.
	t.Run("releases the popped element", func(t *testing.T) {
		first, second := new(int), new(int)
		s := buildStack(first, second)
		got, err := s.Pop()
		if err != nil {
			t.Fatalf("Pop() error = %v, want nil", err)
		}
		if got != second {
			t.Errorf("Pop() returned the wrong pointer")
		}
		assertStack(t, s, []*int{first})
		assertReleased(t, s)
	})
}

func TestStackPeek(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
	}{
		{"one", []int{10}, 10},
		{"two", []int{10, 20}, 20},
		{"many", []int{10, 20, 30}, 30},
		{"duplicates", []int{10, 10}, 10},
		{"zero value on top", []int{10, 0}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := buildStack(tc.start...)
			got, err := s.Peek()
			if err != nil {
				t.Fatalf("Peek() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Peek() = %v, want %v", got, tc.want)
			}
			assertStack(t, s, tc.start)
		})
	}

	t.Run("twice returns the same element", func(t *testing.T) {
		s := buildStack(10, 20)
		first, err := s.Peek()
		if err != nil {
			t.Fatalf("Peek() error = %v, want nil", err)
		}
		second, err := s.Peek()
		if err != nil {
			t.Fatalf("Peek() again error = %v, want nil", err)
		}
		if first != second {
			t.Errorf("Peek() returned %v then %v, want the same element twice", first, second)
		}
		assertStack(t, s, []int{10, 20})
	})
}

func TestStackToSlice(t *testing.T) {
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
			s := buildStack(tc.start...)
			if got := s.ToSlice(); !slices.Equal(got, tc.start) {
				t.Errorf("ToSlice() = %v, want %v bottom to top", got, tc.start)
			}
			assertStack(t, s, tc.start)
		})
	}

	t.Run("empty is non-nil", func(t *testing.T) {
		got := buildStack[int]().ToSlice()
		if got == nil {
			t.Fatal("ToSlice() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
		}
	})

	// Returning the backing slice itself would pass every case above and hand the
	// caller a window onto the stack's own storage.
	t.Run("result is a copy", func(t *testing.T) {
		start := []int{10, 20, 30}
		s := buildStack(start...)
		got := s.ToSlice()
		got[0] = 99
		assertStack(t, s, start)
	})
}

func TestStackString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "nil"},
		{"one", []int{10}, "10 -> nil"},
		{"two", []int{10, 20}, "20 -> 10 -> nil"},
		{"many", []int{10, 20, 30}, "30 -> 20 -> 10 -> nil"},
		{"zero value", []int{0}, "0 -> nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := buildStack(tc.start...)
			if got := s.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
			assertStack(t, s, tc.start)
		})
	}
}

// TestStackPrintStack delegates to String by design - printing the rendered stack is
// the behavior under test - so it goes red if String is broken.
func TestStackPrintStack(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "nil\n"},
		{"one", []int{10}, "10 -> nil\n"},
		{"many", []int{10, 20, 30}, "30 -> 20 -> 10 -> nil\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := buildStack(tc.start...)
			got := captureStdout(t, s.PrintStack)
			if got != tc.want {
				t.Errorf("PrintStack() wrote %q, want %q", got, tc.want)
			}
			assertStack(t, s, tc.start)
		})
	}
}

func ExampleStack() {
	s := New[int]()
	s.Push(10)
	s.Push(20)
	s.Push(30)
	s.PrintStack()

	top, _ := s.Pop()
	fmt.Println(top, s.Len())
	// Output:
	// 30 -> 20 -> 10 -> nil
	// 30 2
}
