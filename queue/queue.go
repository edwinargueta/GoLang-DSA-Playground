// Package queue implements FIFO queues from first principles, ring-backed and node-backed.
package queue

import (
	"errors"
	"fmt"
	"strings"
)

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
	return &Queue[T]{}
}

// Len returns the number of elements. O(1) time, O(1) space.
func (q *Queue[T]) Len() int {
	return q.size
}

// IsEmpty reports whether the queue holds no elements. O(1) time, O(1) space.
func (q *Queue[T]) IsEmpty() bool {
	return q.size == 0
}

// Enqueue adds value at the back, growing the ring if it is full. O(1) amortized time, O(1) space.
func (q *Queue[T]) Enqueue(value T) {
	if q.size == len(q.items) {
		q.grow()
	}
	q.items[(q.head+q.size)%len(q.items)] = value
	q.size++
}

// Dequeue removes and returns the front, zeroing the slot it vacates; ErrEmptyQueue if empty. O(1) time, O(1) space.
func (q *Queue[T]) Dequeue() (T, error) {
	if q.size == 0 {
		var zero T
		return zero, ErrEmptyQueue
	}
	var zero T
	value := q.items[q.head]
	q.items[q.head] = zero
	q.head = (q.head + 1) % len(q.items)
	q.size--
	return value, nil
}

// Peek returns the front without removing it; ErrEmptyQueue if empty. O(1) time, O(1) space.
func (q *Queue[T]) Peek() (T, error) {
	if q.size == 0 {
		var zero T
		return zero, ErrEmptyQueue
	}
	return q.items[q.head], nil
}

// ToSlice returns the values front to back, empty and non-nil for an empty queue. O(n) time, O(n) space.
func (q *Queue[T]) ToSlice() []T {
	values := make([]T, q.size)
	for i := range values {
		values[i] = q.items[(q.head+i)%len(q.items)]
	}
	return values
}

// String renders the queue front first as "10 -> 20 -> 30 -> nil". O(n) time, O(n) space.
func (q *Queue[T]) String() string {
	var b strings.Builder
	for i := 0; i < q.size; i++ {
		fmt.Fprintf(&b, "%v -> ", q.items[(q.head+i)%len(q.items)])
	}
	b.WriteString("nil")
	return b.String()
}

// PrintQueue writes String followed by a newline to standard output. O(n) time, O(n) space.
func (q *Queue[T]) PrintQueue() {
	fmt.Println(q.String())
}

// grow moves the live window to the front of a ring twice the size, minimum 4. O(n) time, O(n) space.
func (q *Queue[T]) grow() {
	items := make([]T, max(4, 2*len(q.items)))
	for i := 0; i < q.size; i++ {
		items[i] = q.items[(q.head+i)%len(q.items)]
	}
	q.items = items
	q.head = 0
}
