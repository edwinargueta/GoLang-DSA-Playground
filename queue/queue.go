// Package queue implements FIFO queues from first principles, ring-backed and node-backed.
package queue

import "errors"

var (
	// ErrEmptyQueue is returned by any operation that needs an element and has none.
	// Both queues in this package share it.
	ErrEmptyQueue = errors.New("queue: queue is empty")
)

// Queue is a first-in-first-out collection backed by a slice used as a ring: head is
// the index of the front, size is how many elements are live, and the back sits at
// (head + size) % len(items). The live window wraps around the end of the array
// rather than stopping there, which is what makes both ends O(1) on one allocation.
//
// A queue is the case a slice is bad at head-on. Dequeue by shifting — copy(items,
// items[1:]) — is O(n) every time. Dequeue by reslicing — items = items[1:] — is O(1)
// but abandons the front of the array permanently, so a queue that is enqueued and
// dequeued a million times allocates a million cells to hold three. The ring reuses
// them instead.
//
// Storing size rather than a tail index is the other half of the design. With two
// indices, head == tail describes both an empty ring and a full one and there is no
// way to tell them apart without wasting a cell; a count has no such ambiguity.
//
// When the ring fills, Enqueue allocates a larger one and copies the live window into
// it, which is what makes Enqueue O(1) amortized rather than O(1). Dequeue zeroes the
// slot it vacates, so every cell outside the live window holds the zero value and a
// dequeued element is not kept alive by the ring — invisible for a queue of ints,
// unbounded for a queue of pointers.
//
// Duplicates are kept and T is any, because a queue never compares its elements. A
// zero T is a legitimate element, so Dequeue and Peek report emptiness through an
// error — never by returning the zero value alone.
//
// The zero Queue is a valid empty queue, with a nil ring the first Enqueue allocates;
// New is a convenience.
type Queue[T any] struct {
	items []T
	head  int
	size  int
}

// New returns an empty ring-backed queue. O(1) time, O(1) space.
func New[T any]() *Queue[T] {
	panic("not implemented")
}

// Len returns the number of elements. O(1) time, O(1) space.
func (q *Queue[T]) Len() int {
	panic("not implemented")
}

// IsEmpty reports whether the queue holds no elements. O(1) time, O(1) space.
func (q *Queue[T]) IsEmpty() bool {
	panic("not implemented")
}

// Enqueue adds value at the back, growing the ring if it is full. O(1) amortized time, O(1) space.
func (q *Queue[T]) Enqueue(value T) {
	panic("not implemented")
}

// Dequeue removes and returns the front, zeroing the slot it vacates; ErrEmptyQueue if empty. O(1) time, O(1) space.
func (q *Queue[T]) Dequeue() (T, error) {
	panic("not implemented")
}

// Peek returns the front without removing it; ErrEmptyQueue if empty. O(1) time, O(1) space.
func (q *Queue[T]) Peek() (T, error) {
	panic("not implemented")
}

// ToSlice returns the values front to back, empty and non-nil for an empty queue. O(n) time, O(n) space.
func (q *Queue[T]) ToSlice() []T {
	panic("not implemented")
}

// String renders the queue front first as "10 -> 20 -> 30 -> nil". O(n) time, O(n) space.
func (q *Queue[T]) String() string {
	panic("not implemented")
}

// PrintQueue writes String followed by a newline to standard output. O(n) time, O(n) space.
func (q *Queue[T]) PrintQueue() {
	panic("not implemented")
}
