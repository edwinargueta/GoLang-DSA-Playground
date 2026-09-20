package stack

import "cmp"

// MinStack is a slice-backed stack that also answers "what is the smallest value you
// hold" in constant time, by keeping a second stack of running minima alongside the
// values.
//
// The invariant is that the two slices always have the same length and mins[i] is the
// smallest of values[:i+1]. Push therefore appends to both — the new value, and the
// smaller of it and the current minimum — and Pop removes from both. Min is then a
// read of the last element of mins, with no scan.
//
// Pushing a minimum only when it is a new one saves space and is the classic way to
// get this wrong: pop the single recorded copy of a minimum that was pushed twice and
// the stack forgets a value it still holds. Recording one minimum per element costs
// O(n) extra space and has no such case, which is the trade this type takes.
//
// Popping is where the invariant is usually broken, by shrinking values and leaving
// mins alone. Nothing about the next Push or Peek reveals it; only Min goes wrong,
// and only later.
//
// Duplicates are kept, including duplicate minima. T is cmp.Ordered because this
// stack compares its elements, unlike Stack and LinkedStack. A zero T is a legitimate
// element, so Pop, Peek and Min report emptiness through an error — never by
// returning the zero value alone.
//
// The zero MinStack is a valid empty stack; NewMin is a convenience.
type MinStack[T cmp.Ordered] struct {
	values []T
	mins   []T
}

// NewMin returns an empty minimum-tracking stack. O(1) time, O(1) space.
func NewMin[T cmp.Ordered]() *MinStack[T] {
	panic("not implemented")
}

// Len returns the number of elements. O(1) time, O(1) space.
func (s *MinStack[T]) Len() int {
	panic("not implemented")
}

// IsEmpty reports whether the stack holds no elements. O(1) time, O(1) space.
func (s *MinStack[T]) IsEmpty() bool {
	panic("not implemented")
}

// Push adds value at the top. O(1) amortized time, O(1) space.
func (s *MinStack[T]) Push(value T) {
	panic("not implemented")
}

// Pop removes and returns the top, zeroing the slots it vacates; ErrEmptyStack if empty. O(1) time, O(1) space.
func (s *MinStack[T]) Pop() (T, error) {
	panic("not implemented")
}

// Peek returns the top without removing it; ErrEmptyStack if empty. O(1) time, O(1) space.
func (s *MinStack[T]) Peek() (T, error) {
	panic("not implemented")
}

// Min returns the smallest value the stack holds; ErrEmptyStack if empty. O(1) time, O(1) space.
func (s *MinStack[T]) Min() (T, error) {
	panic("not implemented")
}

// ToSlice returns the values bottom to top, empty and non-nil for an empty stack. O(n) time, O(n) space.
func (s *MinStack[T]) ToSlice() []T {
	panic("not implemented")
}

// String renders the stack top first as "30 -> 20 -> 10 -> nil". O(n) time, O(n) space.
func (s *MinStack[T]) String() string {
	panic("not implemented")
}

// PrintStack writes String followed by a newline to standard output. O(n) time, O(n) space.
func (s *MinStack[T]) PrintStack() {
	panic("not implemented")
}
