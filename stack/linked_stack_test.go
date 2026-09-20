// White-box tests for the node-backed stack.
//
//	go test ./stack/ -run TestLinkedStack -v     # this type
//	go test ./stack/ -run TestLinkedStackPop -v  # one method
//
// Test names are prefixed with the type because three stacks share this package and
// Go test function names are package-scoped; captureStdout lives in stack_test.go,
// where one declaration serves all three suites.
//
// The oracle walks the chain from top, so no method is ever used to verify another.
// walkDown is bounded by maxNodes: a stack wired into a cycle must fail the suite
// rather than hang it. The file is package stack, not package stack_test, so those
// helpers can read top, next and size while the fields stay unexported.
//
// Two deliberate exceptions, each noted again above the test itself:
//   - TestLinkedStackShape guards the shape of the structure rather than a method.
//   - ExampleLinkedStack exercises NewLinked, Push, PrintStack, Pop and Len together,
//     because a runnable example is by definition an integration.
package stack

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
)

// maxNodes bounds every walking helper, so a chain wired into a cycle fails the suite
// instead of hanging it.
const maxNodes = 1000

// walkDown collects values by following next from top, independent of any method.
func walkDown[T comparable](top *node[T], limit int) []T {
	var values []T
	for n := top; n != nil; n = n.next {
		if len(values) > limit {
			panic("walked past the limit - the chain has a cycle")
		}
		values = append(values, n.value)
	}
	return values
}

// reversed returns a copy of values back to front, so a want written bottom to top
// can be compared against a walk that runs top to bottom.
func reversed[T comparable](values []T) []T {
	out := make([]T, len(values))
	for i, v := range values {
		out[len(values)-1-i] = v
	}
	return out
}

// buildLinkedStack returns a stack holding values bottom to top, wiring the nodes by
// hand rather than calling Push, so every test in this file is independent of the
// implementation and the methods can be written in any order.
func buildLinkedStack[T comparable](values ...T) *LinkedStack[T] {
	s := &LinkedStack[T]{}
	for _, v := range values {
		s.top = &node[T]{value: v, next: s.top}
		s.size++
	}
	return s
}

// assertLinkedStack checks every structural invariant at once: the chain read from
// top, the size counter, a nil top when the stack is empty, and a nil next on the
// bottom node. It reads s.size directly rather than calling Len, so it never borrows
// a method to judge another. Call it after every mutation, and after every rejected
// one too - a call that returns an error must leave the stack untouched. want is
// written bottom to top, matching buildLinkedStack and ToSlice.
func assertLinkedStack[T comparable](t *testing.T, s *LinkedStack[T], want []T) {
	t.Helper()

	if got, w := walkDown(s.top, maxNodes), reversed(want); !slices.Equal(got, w) {
		t.Errorf("chain from top = %v, want %v", got, w)
	}
	if s.size != len(want) {
		t.Errorf("size field = %d, want %d", s.size, len(want))
	}
	if len(want) == 0 {
		if s.top != nil {
			t.Errorf("top = node(%v), want nil on an empty stack", s.top.value)
		}
		return
	}
	if s.top == nil {
		t.Fatalf("top = nil, want a node holding %v", want[len(want)-1])
	}
	bottom := s.top
	for bottom.next != nil {
		bottom = bottom.next
	}
	if bottom.value != want[0] {
		t.Errorf("bottom node = %v, want %v", bottom.value, want[0])
	}
}

// TestLinkedStackShape pins the shape of the structure rather than any one method.
// A stack touches one end only, so there is no tail pointer here and the node has no
// prev - the moment either appears, this is a different structure.
func TestLinkedStackShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  reflect.Type
		want []string
	}{
		{"stack", reflect.TypeOf(LinkedStack[int]{}), []string{"top", "size"}},
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

// TestLinkedStackEmpty is the case most implementations get wrong, so every method
// gets a subtest against a freshly built empty stack.
func TestLinkedStackEmpty(t *testing.T) {
	t.Run("Len", func(t *testing.T) {
		if got := buildLinkedStack[int]().Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
	})
	t.Run("IsEmpty", func(t *testing.T) {
		if got := buildLinkedStack[int]().IsEmpty(); !got {
			t.Errorf("IsEmpty() = false, want true")
		}
	})
	t.Run("Push", func(t *testing.T) {
		s := buildLinkedStack[int]()
		s.Push(10)
		assertLinkedStack(t, s, []int{10})
	})
	t.Run("Pop", func(t *testing.T) {
		s := buildLinkedStack[int]()
		got, err := s.Pop()
		if !errors.Is(err, ErrEmptyStack) {
			t.Errorf("Pop() error = %v, want ErrEmptyStack", err)
		}
		if got != 0 {
			t.Errorf("Pop() = %d, want the zero value", got)
		}
		assertLinkedStack(t, s, nil)
	})
	t.Run("Peek", func(t *testing.T) {
		s := buildLinkedStack[int]()
		got, err := s.Peek()
		if !errors.Is(err, ErrEmptyStack) {
			t.Errorf("Peek() error = %v, want ErrEmptyStack", err)
		}
		if got != 0 {
			t.Errorf("Peek() = %d, want the zero value", got)
		}
		assertLinkedStack(t, s, nil)
	})
	t.Run("ToSlice", func(t *testing.T) {
		got := buildLinkedStack[int]().ToSlice()
		if got == nil {
			t.Fatal("ToSlice() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
		}
	})
	t.Run("String", func(t *testing.T) {
		if got := buildLinkedStack[int]().String(); got != "nil" {
			t.Errorf("String() = %q, want %q", got, "nil")
		}
	})
}

func TestLinkedStackNewLinked(t *testing.T) {
	s := NewLinked[int]()
	if s == nil {
		t.Fatal("NewLinked() = nil, want an empty stack")
	}
	assertLinkedStack(t, s, nil)
}

func TestLinkedStackLen(t *testing.T) {
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
			if got := buildLinkedStack(tc.start...).Len(); got != len(tc.start) {
				t.Errorf("Len() = %d, want %d", got, len(tc.start))
			}
		})
	}
}

func TestLinkedStackIsEmpty(t *testing.T) {
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
			if got := buildLinkedStack(tc.start...).IsEmpty(); got != tc.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLinkedStackPush(t *testing.T) {
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
			s := buildLinkedStack(tc.start...)
			s.Push(tc.value)
			assertLinkedStack(t, s, tc.want)
		})
	}

	t.Run("repeated onto empty", func(t *testing.T) {
		s := buildLinkedStack[int]()
		want := []int{10, 20, 30}
		for i, v := range want {
			s.Push(v)
			assertLinkedStack(t, s, want[:i+1])
		}
	})
}

func TestLinkedStackPop(t *testing.T) {
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
			s := buildLinkedStack(tc.start...)
			got, err := s.Pop()
			if err != nil {
				t.Fatalf("Pop() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Pop() = %v, want %v", got, tc.want)
			}
			assertLinkedStack(t, s, tc.rest)
		})
	}

	t.Run("drains last in first out", func(t *testing.T) {
		start := []int{10, 20, 30}
		s := buildLinkedStack(start...)
		for i := len(start) - 1; i >= 0; i-- {
			got, err := s.Pop()
			if err != nil {
				t.Fatalf("Pop() at %d error = %v, want nil", i, err)
			}
			if got != start[i] {
				t.Errorf("Pop() at %d = %v, want %v", i, got, start[i])
			}
			assertLinkedStack(t, s, start[:i])
		}
		if _, err := s.Pop(); !errors.Is(err, ErrEmptyStack) {
			t.Errorf("Pop() on the drained stack error = %v, want ErrEmptyStack", err)
		}
	})

	// Draining and refilling is what catches a top left pointing at a node that was
	// already removed: nothing before the next Push reveals it.
	t.Run("drain then refill", func(t *testing.T) {
		s := buildLinkedStack(10, 20)
		for i := 0; i < 2; i++ {
			if _, err := s.Pop(); err != nil {
				t.Fatalf("Pop() #%d error = %v, want nil", i, err)
			}
		}
		assertLinkedStack(t, s, nil)
		s.Push(30)
		assertLinkedStack(t, s, []int{30})
	})
}

func TestLinkedStackPeek(t *testing.T) {
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
			s := buildLinkedStack(tc.start...)
			got, err := s.Peek()
			if err != nil {
				t.Fatalf("Peek() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Peek() = %v, want %v", got, tc.want)
			}
			assertLinkedStack(t, s, tc.start)
		})
	}

	t.Run("twice returns the same element", func(t *testing.T) {
		s := buildLinkedStack(10, 20)
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
		assertLinkedStack(t, s, []int{10, 20})
	})
}

func TestLinkedStackToSlice(t *testing.T) {
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
			s := buildLinkedStack(tc.start...)
			if got := s.ToSlice(); !slices.Equal(got, tc.start) {
				t.Errorf("ToSlice() = %v, want %v bottom to top", got, tc.start)
			}
			assertLinkedStack(t, s, tc.start)
		})
	}

	t.Run("empty is non-nil", func(t *testing.T) {
		got := buildLinkedStack[int]().ToSlice()
		if got == nil {
			t.Fatal("ToSlice() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
		}
	})
}

func TestLinkedStackString(t *testing.T) {
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
			s := buildLinkedStack(tc.start...)
			if got := s.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
			assertLinkedStack(t, s, tc.start)
		})
	}
}

// TestLinkedStackPrintStack delegates to String by design - printing the rendered
// stack is the behavior under test - so it goes red if String is broken.
func TestLinkedStackPrintStack(t *testing.T) {
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
			s := buildLinkedStack(tc.start...)
			got := captureStdout(t, s.PrintStack)
			if got != tc.want {
				t.Errorf("PrintStack() wrote %q, want %q", got, tc.want)
			}
			assertLinkedStack(t, s, tc.start)
		})
	}
}

func ExampleLinkedStack() {
	s := NewLinked[string]()
	s.Push("a")
	s.Push("b")
	s.Push("c")
	s.PrintStack()

	top, _ := s.Pop()
	fmt.Println(top, s.Len())
	// Output:
	// c -> b -> a -> nil
	// c 2
}
