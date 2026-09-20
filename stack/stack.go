// Package stack implements LIFO stacks from first principles, slice-backed and node-backed.
package stack

import "errors"

var (
	// ErrEmptyStack is returned by any operation that needs an element and has none.
	// All three stacks in this package share it.
	ErrEmptyStack = errors.New("stack: stack is empty")
)

// Stack is a last-in-first-out collection backed by a slice: Push appends and Pop
// reslices, so both touch only the end, which is the end a slice is cheap at.
//
// Duplicates are kept and nothing is ordered — this is a sequence with one open end.
// T is any because a stack never compares its elements to each other. Only MinStack
// in this package needs cmp.Ordered, which is why it is a separate type rather than a
// Min method here.
//
// Push is O(1) amortized rather than O(1): when the backing array fills, append
// allocates a larger one and copies, so one push in every doubling pays O(n) and the
// rest pay nothing. LinkedStack pays a small constant on every push instead and never
// spikes. That is the whole trade between the two, along with locality — this one
// holds its elements in a single contiguous run, the other scatters them across as
// many allocations as it has elements.
//
// Pop zeroes the slot it vacates before shrinking the length. The popped element sits
// above the length but still inside the capacity, so the backing array would
// otherwise keep it alive for as long as the stack does: invisible for a stack of
// ints, unbounded for a stack of pointers.
//
// ToSlice reads bottom to top, the order the elements were pushed in, while String
// renders top first, the order they will come back out. A zero T is a legitimate
// element, so Pop and Peek report emptiness through an error — never by returning the
// zero value alone.
//
// The zero Stack is a valid empty stack; New is a convenience.
type Stack[T any] struct {
	items []T
}

// New returns an empty slice-backed stack. O(1) time, O(1) space.
func New[T any]() *Stack[T] {
	panic("not implemented")
}

// Len returns the number of elements. O(1) time, O(1) space.
func (s *Stack[T]) Len() int {
	panic("not implemented")
}

// IsEmpty reports whether the stack holds no elements. O(1) time, O(1) space.
func (s *Stack[T]) IsEmpty() bool {
	panic("not implemented")
}

// Push adds value at the top. O(1) amortized time, O(1) space.
func (s *Stack[T]) Push(value T) {
	panic("not implemented")
}

// Pop removes and returns the top, zeroing the slot it vacates; ErrEmptyStack if empty. O(1) time, O(1) space.
func (s *Stack[T]) Pop() (T, error) {
	panic("not implemented")
}

// Peek returns the top without removing it; ErrEmptyStack if empty. O(1) time, O(1) space.
func (s *Stack[T]) Peek() (T, error) {
	panic("not implemented")
}

// ToSlice returns the values bottom to top, empty and non-nil for an empty stack. O(n) time, O(n) space.
func (s *Stack[T]) ToSlice() []T {
	panic("not implemented")
}

// String renders the stack top first as "30 -> 20 -> 10 -> nil". O(n) time, O(n) space.
func (s *Stack[T]) String() string {
	panic("not implemented")
}

// PrintStack writes String followed by a newline to standard output. O(n) time, O(n) space.
func (s *Stack[T]) PrintStack() {
	panic("not implemented")
}
