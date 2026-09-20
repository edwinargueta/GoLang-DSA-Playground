package queue

// LinkedQueue is a first-in-first-out collection backed by a chain of nodes, holding
// head for the front and tail for the back so that both ends are O(1) without any
// arithmetic.
//
// This is the structure the singly linked list in this repo is bad at and this one is
// not, for one reason: a queue only ever removes from the front. Removing from the
// back would need the node before the tail, which no forward-only chain can give you
// in constant time — that is what the doubly linked list is for. A queue never asks.
//
// The invariants are that tail.next is always nil, that head is nil exactly when tail
// is, and that size matches the chain. The characteristic bug is a stale tail: drain
// the queue through Dequeue, leave tail pointing at the node that was just removed,
// and the next Enqueue links a new node onto garbage and loses it. Nothing before
// that next Enqueue reveals it, which is why the suite asserts tail directly rather
// than reading values back out.
//
// Against Queue: no reallocation and no amortization, so every Enqueue costs the
// same, but one allocation and one pointer-chase per element and no locality. The
// ring wins on throughput, this one on predictability.
//
// Duplicates are kept and T is any. A zero T is a legitimate element, so Dequeue and
// Peek report emptiness through an error — never by returning the zero value alone.
//
// The zero LinkedQueue is a valid empty queue; NewLinked is a convenience.
type LinkedQueue[T any] struct {
	head *node[T]
	tail *node[T]
	size int
}

// NewLinked returns an empty node-backed queue. O(1) time, O(1) space.
func NewLinked[T any]() *LinkedQueue[T] {
	panic("not implemented")
}

// Len returns the number of elements. O(1) time, O(1) space.
func (q *LinkedQueue[T]) Len() int {
	panic("not implemented")
}

// IsEmpty reports whether the queue holds no elements. O(1) time, O(1) space.
func (q *LinkedQueue[T]) IsEmpty() bool {
	panic("not implemented")
}

// Enqueue adds value at the back. O(1) time, O(1) space.
func (q *LinkedQueue[T]) Enqueue(value T) {
	panic("not implemented")
}

// Dequeue removes and returns the front; ErrEmptyQueue if empty. O(1) time, O(1) space.
func (q *LinkedQueue[T]) Dequeue() (T, error) {
	panic("not implemented")
}

// Peek returns the front without removing it; ErrEmptyQueue if empty. O(1) time, O(1) space.
func (q *LinkedQueue[T]) Peek() (T, error) {
	panic("not implemented")
}

// ToSlice returns the values front to back, empty and non-nil for an empty queue. O(n) time, O(n) space.
func (q *LinkedQueue[T]) ToSlice() []T {
	panic("not implemented")
}

// String renders the queue front first as "10 -> 20 -> 30 -> nil". O(n) time, O(n) space.
func (q *LinkedQueue[T]) String() string {
	panic("not implemented")
}

// PrintQueue writes String followed by a newline to standard output. O(n) time, O(n) space.
func (q *LinkedQueue[T]) PrintQueue() {
	panic("not implemented")
}
