// White-box tests for the minimum-tracking stack.
//
//	go test ./stack/ -run TestMinStack -v     # this type
//	go test ./stack/ -run TestMinStackPop -v  # one method
//
// Test names are prefixed with the type because three stacks share this package and
// Go test function names are package-scoped; captureStdout lives in stack_test.go,
// where one declaration serves all three suites.
//
// The oracle reads both backing slices. assertMinStack recomputes the running minima
// from scratch with runningMin and compares, so a mins slice that has drifted out of
// step with values fails immediately rather than at whichever later Min call happens
// to read the wrong cell. That drift is the characteristic bug of this structure:
// Pop that shrinks values and forgets mins leaves a stack that pushes, peeks and
// pops perfectly and answers Min wrong.
//
// Every mutation test therefore calls assertMinStack rather than checking Min, which
// keeps each test failing only on its own method.
//
// Two deliberate exceptions, each noted again above the test itself:
//   - TestMinStackShape guards the shape of the structure rather than a method.
//   - ExampleMinStack exercises NewMin, Push, Min, Pop and PrintStack together,
//     because a runnable example is by definition an integration.
package stack

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
)

// runningMin returns the smallest of values[:i+1] at each i, which is what the mins
// slice is supposed to hold. Recomputed here so the suite never trusts the structure
// to check itself.
func runningMin[T cmp.Ordered](values []T) []T {
	out := make([]T, len(values))
	for i, v := range values {
		if i > 0 && out[i-1] < v {
			v = out[i-1]
		}
		out[i] = v
	}
	return out
}

// buildMinStack returns a stack holding values bottom to top, filling both backing
// slices directly rather than calling Push, so every test in this file is
// independent of the implementation and the methods can be written in any order.
func buildMinStack[T cmp.Ordered](values ...T) *MinStack[T] {
	return &MinStack[T]{
		values: append([]T(nil), values...),
		mins:   runningMin(values),
	}
}

// assertMinStack checks every structural invariant at once: the elements bottom to
// top, that mins is exactly as long as values, and that every mins[i] is the
// smallest of values[:i+1]. Call it after every mutation, and after every rejected
// one too - a call that returns an error must leave the stack untouched.
func assertMinStack[T cmp.Ordered](t *testing.T, s *MinStack[T], want []T) {
	t.Helper()

	if got := s.values; !slices.Equal(got, want) {
		t.Errorf("values bottom to top = %v, want %v", got, want)
	}
	if len(s.mins) != len(s.values) {
		t.Errorf("len(mins) = %d, len(values) = %d, want them equal - the two stacks have drifted apart", len(s.mins), len(s.values))
		return
	}
	if got, w := s.mins, runningMin(want); !slices.Equal(got, w) {
		t.Errorf("mins = %v, want %v - mins[i] must be the smallest of values[:i+1]", got, w)
	}

	// Both slices are shortened by reslicing, which leaves the popped element and
	// its minimum above the length and still inside the capacity, where the backing
	// array keeps them alive for as long as the stack lives.
	var zero T
	if n := len(s.values); cap(s.values) > n && s.values[:n+1][n] != zero {
		t.Errorf("the cell above the top of values holds %v, want the zero value - Pop left the popped element reachable", s.values[:n+1][n])
	}
	if n := len(s.mins); cap(s.mins) > n && s.mins[:n+1][n] != zero {
		t.Errorf("the cell above the top of mins holds %v, want the zero value - Pop left the popped minimum reachable", s.mins[:n+1][n])
	}
}

// TestMinStackShape pins the shape of the structure rather than any one method. Two
// slices, kept the same length: the second one is the whole idea, and a version that
// stored a single current minimum instead would be a different and broken type.
func TestMinStackShape(t *testing.T) {
	var got []string
	for _, f := range reflect.VisibleFields(reflect.TypeOf(MinStack[int]{})) {
		got = append(got, f.Name)
	}
	if want := []string{"values", "mins"}; !slices.Equal(got, want) {
		t.Errorf("MinStack fields = %v, want %v", got, want)
	}
}

// TestMinStackEmpty is the case most implementations get wrong, so every method gets
// a subtest against a freshly built empty stack.
func TestMinStackEmpty(t *testing.T) {
	t.Run("Len", func(t *testing.T) {
		if got := buildMinStack[int]().Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
	})
	t.Run("IsEmpty", func(t *testing.T) {
		if got := buildMinStack[int]().IsEmpty(); !got {
			t.Errorf("IsEmpty() = false, want true")
		}
	})
	t.Run("Push", func(t *testing.T) {
		s := buildMinStack[int]()
		s.Push(10)
		assertMinStack(t, s, []int{10})
	})
	t.Run("Pop", func(t *testing.T) {
		s := buildMinStack[int]()
		got, err := s.Pop()
		if !errors.Is(err, ErrEmptyStack) {
			t.Errorf("Pop() error = %v, want ErrEmptyStack", err)
		}
		if got != 0 {
			t.Errorf("Pop() = %d, want the zero value", got)
		}
		assertMinStack(t, s, nil)
	})
	t.Run("Peek", func(t *testing.T) {
		s := buildMinStack[int]()
		got, err := s.Peek()
		if !errors.Is(err, ErrEmptyStack) {
			t.Errorf("Peek() error = %v, want ErrEmptyStack", err)
		}
		if got != 0 {
			t.Errorf("Peek() = %d, want the zero value", got)
		}
		assertMinStack(t, s, nil)
	})
	t.Run("Min", func(t *testing.T) {
		s := buildMinStack[int]()
		got, err := s.Min()
		if !errors.Is(err, ErrEmptyStack) {
			t.Errorf("Min() error = %v, want ErrEmptyStack", err)
		}
		if got != 0 {
			t.Errorf("Min() = %d, want the zero value", got)
		}
		assertMinStack(t, s, nil)
	})
	t.Run("ToSlice", func(t *testing.T) {
		got := buildMinStack[int]().ToSlice()
		if got == nil {
			t.Fatal("ToSlice() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
		}
	})
	t.Run("String", func(t *testing.T) {
		if got := buildMinStack[int]().String(); got != "nil" {
			t.Errorf("String() = %q, want %q", got, "nil")
		}
	})
}

func TestMinStackNewMin(t *testing.T) {
	s := NewMin[int]()
	if s == nil {
		t.Fatal("NewMin() = nil, want an empty stack")
	}
	assertMinStack(t, s, nil)
}

func TestMinStackLen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
	}{
		{"empty", nil},
		{"one", []int{10}},
		{"two", []int{10, 20}},
		{"many", []int{30, 10, 20, 40}},
		{"duplicates", []int{10, 10, 10}},
		{"zero values", []int{0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildMinStack(tc.start...).Len(); got != len(tc.start) {
				t.Errorf("Len() = %d, want %d", got, len(tc.start))
			}
		})
	}
}

func TestMinStackIsEmpty(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  bool
	}{
		{"empty", nil, true},
		{"one", []int{10}, false},
		{"many", []int{30, 10, 20}, false},
		{"one zero value", []int{0}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildMinStack(tc.start...).IsEmpty(); got != tc.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMinStackPush(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		value int
		want  []int
	}{
		{"onto empty", nil, 10, []int{10}},
		{"above the current minimum", []int{10}, 20, []int{10, 20}},
		{"below the current minimum", []int{20}, 10, []int{20, 10}},
		{"equal to the current minimum", []int{10, 20}, 10, []int{10, 20, 10}},
		{"a new minimum twice over", []int{30, 20}, 10, []int{30, 20, 10}},
		{"zero value below a positive minimum", []int{10}, 0, []int{10, 0}},
		{"negative below zero", []int{0}, -5, []int{0, -5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := buildMinStack(tc.start...)
			s.Push(tc.value)
			assertMinStack(t, s, tc.want)
		})
	}

	t.Run("repeated onto empty", func(t *testing.T) {
		s := buildMinStack[int]()
		want := []int{30, 10, 20, 10, 40}
		for i, v := range want {
			s.Push(v)
			assertMinStack(t, s, want[:i+1])
		}
	})
}

func TestMinStackPop(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
		rest  []int
	}{
		{"one", []int{10}, 10, nil},
		{"the top is not the minimum", []int{10, 20}, 20, []int{10}},
		{"the top is the minimum", []int{20, 10}, 10, []int{20}},
		{"one of two equal minima", []int{10, 20, 10}, 10, []int{10, 20}},
		{"many", []int{30, 10, 20}, 20, []int{30, 10}},
		{"zero value on top", []int{10, 0}, 0, []int{10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := buildMinStack(tc.start...)
			got, err := s.Pop()
			if err != nil {
				t.Fatalf("Pop() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Pop() = %v, want %v", got, tc.want)
			}
			assertMinStack(t, s, tc.rest)
		})
	}

	// Popping the minimum has to restore the one that was in force beneath it. This
	// is where a mins stack that only records new minima loses a value it still holds.
	t.Run("drains and the minimum climbs back", func(t *testing.T) {
		start := []int{30, 10, 20, 10}
		s := buildMinStack(start...)
		for i := len(start) - 1; i >= 0; i-- {
			got, err := s.Pop()
			if err != nil {
				t.Fatalf("Pop() at %d error = %v, want nil", i, err)
			}
			if got != start[i] {
				t.Errorf("Pop() at %d = %v, want %v", i, got, start[i])
			}
			assertMinStack(t, s, start[:i])
		}
		if _, err := s.Pop(); !errors.Is(err, ErrEmptyStack) {
			t.Errorf("Pop() on the drained stack error = %v, want ErrEmptyStack", err)
		}
	})

	t.Run("drain then refill", func(t *testing.T) {
		s := buildMinStack(20, 10)
		for i := 0; i < 2; i++ {
			if _, err := s.Pop(); err != nil {
				t.Fatalf("Pop() #%d error = %v, want nil", i, err)
			}
		}
		assertMinStack(t, s, nil)
		s.Push(30)
		assertMinStack(t, s, []int{30})
	})
}

func TestMinStackPeek(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
	}{
		{"one", []int{10}, 10},
		{"the top is not the minimum", []int{10, 20}, 20},
		{"the top is the minimum", []int{20, 10}, 10},
		{"duplicates", []int{10, 10}, 10},
		{"zero value on top", []int{10, 0}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := buildMinStack(tc.start...)
			got, err := s.Peek()
			if err != nil {
				t.Fatalf("Peek() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Peek() = %v, want %v", got, tc.want)
			}
			assertMinStack(t, s, tc.start)
		})
	}
}

func TestMinStackMin(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
	}{
		{"one", []int{10}, 10},
		{"minimum at the bottom", []int{10, 20, 30}, 10},
		{"minimum on top", []int{30, 20, 10}, 10},
		{"minimum in the middle", []int{30, 10, 20}, 10},
		{"all equal", []int{10, 10, 10}, 10},
		{"two equal minima", []int{10, 20, 10}, 10},
		{"zero value is the minimum", []int{10, 0, 20}, 0},
		{"negative minimum", []int{10, -5, 20}, -5},
		{"all negative", []int{-10, -20, -5}, -20},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := buildMinStack(tc.start...)
			got, err := s.Min()
			if err != nil {
				t.Fatalf("Min() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Min() = %v, want %v", got, tc.want)
			}
			assertMinStack(t, s, tc.start)
		})
	}

	t.Run("reads the last cell, not the first", func(t *testing.T) {
		// An implementation that returns mins[0] instead of the top of mins agrees
		// with every case above where the minimum never changes. Here it does.
		s := buildMinStack(30, 20, 10)
		got, err := s.Min()
		if err != nil {
			t.Fatalf("Min() error = %v, want nil", err)
		}
		if got != 10 {
			t.Errorf("Min() = %v, want 10", got)
		}
	})
}

func TestMinStackToSlice(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
	}{
		{"one", []int{10}},
		{"two", []int{20, 10}},
		{"many", []int{30, 10, 20, 40}},
		{"duplicates", []int{10, 10, 10}},
		{"zero values", []int{0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := buildMinStack(tc.start...)
			if got := s.ToSlice(); !slices.Equal(got, tc.start) {
				t.Errorf("ToSlice() = %v, want %v bottom to top", got, tc.start)
			}
			assertMinStack(t, s, tc.start)
		})
	}

	t.Run("empty is non-nil", func(t *testing.T) {
		got := buildMinStack[int]().ToSlice()
		if got == nil {
			t.Fatal("ToSlice() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
		}
	})

	// Returning the values slice itself would pass every case above and hand the
	// caller a window onto the stack's own storage - and onto the half of it that
	// mins is kept in step with.
	t.Run("result is a copy", func(t *testing.T) {
		start := []int{30, 10, 20}
		s := buildMinStack(start...)
		got := s.ToSlice()
		got[0] = 99
		assertMinStack(t, s, start)
	})
}

func TestMinStackString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "nil"},
		{"one", []int{10}, "10 -> nil"},
		{"two", []int{20, 10}, "10 -> 20 -> nil"},
		{"many", []int{30, 10, 20}, "20 -> 10 -> 30 -> nil"},
		{"zero value", []int{0}, "0 -> nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := buildMinStack(tc.start...)
			if got := s.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
			assertMinStack(t, s, tc.start)
		})
	}
}

// TestMinStackPrintStack delegates to String by design - printing the rendered stack
// is the behavior under test - so it goes red if String is broken.
func TestMinStackPrintStack(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "nil\n"},
		{"one", []int{10}, "10 -> nil\n"},
		{"many", []int{30, 10, 20}, "20 -> 10 -> 30 -> nil\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := buildMinStack(tc.start...)
			got := captureStdout(t, s.PrintStack)
			if got != tc.want {
				t.Errorf("PrintStack() wrote %q, want %q", got, tc.want)
			}
			assertMinStack(t, s, tc.start)
		})
	}
}

func ExampleMinStack() {
	s := NewMin[int]()
	for _, v := range []int{30, 10, 20} {
		s.Push(v)
	}
	s.PrintStack()

	low, _ := s.Min()
	fmt.Println("min", low)

	s.Pop()
	s.Pop()
	low, _ = s.Min()
	fmt.Println("min", low)
	// Output:
	// 20 -> 10 -> 30 -> nil
	// min 10
	// min 30
}
