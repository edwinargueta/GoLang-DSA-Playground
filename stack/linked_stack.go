package stack

// LinkedStack is a last-in-first-out collection backed by a chain of nodes, with top
// pointing at the most recently pushed one and every node pointing at the cell
// beneath it.
//
// It is the same contract as Stack and the same complexities on paper, but the costs
// are spent differently. Push allocates one node and relinks; there is no backing
// array, so no reallocation and no amortization — every push costs the same, which is
// what you want when a latency spike matters more than throughput. Against that: one
// allocation and one pointer-chase per element, no locality, and a chain of n nodes
// where the slice version has one run of n values. For most workloads the slice
// version wins; the point of writing both is to know why.
//
// There is no tail pointer and none is needed: a stack only ever touches one end, and
// that end is the head of the chain. The bottom node's next is nil, and top is nil
// exactly when size is zero.
//
// Duplicates are kept, T is any, and ToSlice reads bottom to top while String renders
// top first — all as in Stack. A zero T is a legitimate element, so Pop and Peek
// report emptiness through an error.
//
// The zero LinkedStack is a valid empty stack; NewLinked is a convenience.
type LinkedStack[T any] struct {
	top  *node[T]
	size int
}

// NewLinked returns an empty node-backed stack. O(1) time, O(1) space.
func NewLinked[T any]() *LinkedStack[T] {
	panic("not implemented")
}

// Len returns the number of elements. O(1) time, O(1) space.
func (s *LinkedStack[T]) Len() int {
	panic("not implemented")
}

// IsEmpty reports whether the stack holds no elements. O(1) time, O(1) space.
func (s *LinkedStack[T]) IsEmpty() bool {
	panic("not implemented")
}

// Push adds value at the top. O(1) time, O(1) space.
func (s *LinkedStack[T]) Push(value T) {
	panic("not implemented")
}

// Pop removes and returns the top; ErrEmptyStack if empty. O(1) time, O(1) space.
func (s *LinkedStack[T]) Pop() (T, error) {
	panic("not implemented")
}

// Peek returns the top without removing it; ErrEmptyStack if empty. O(1) time, O(1) space.
func (s *LinkedStack[T]) Peek() (T, error) {
	panic("not implemented")
}

// ToSlice returns the values bottom to top, empty and non-nil for an empty stack. O(n) time, O(n) space.
func (s *LinkedStack[T]) ToSlice() []T {
	panic("not implemented")
}

// String renders the stack top first as "30 -> 20 -> 10 -> nil". O(n) time, O(n) space.
func (s *LinkedStack[T]) String() string {
	panic("not implemented")
}

// PrintStack writes String followed by a newline to standard output. O(n) time, O(n) space.
func (s *LinkedStack[T]) PrintStack() {
	panic("not implemented")
}
