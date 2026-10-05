package queue

import (
	"fmt"
	"strings"
)

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
	return &LinkedQueue[T]{}
}

// Len returns the number of elements. O(1) time, O(1) space.
func (q *LinkedQueue[T]) Len() int {
	return q.size
}

// IsEmpty reports whether the queue holds no elements. O(1) time, O(1) space.
func (q *LinkedQueue[T]) IsEmpty() bool {
	return q.size == 0
}

// Enqueue adds value at the back. O(1) time, O(1) space.
func (q *LinkedQueue[T]) Enqueue(value T) {
	n := &node[T]{value: value}
	if q.tail == nil {
		q.head = n
	} else {
		q.tail.next = n
	}
	q.tail = n
	q.size++
}

// Dequeue removes and returns the front; ErrEmptyQueue if empty. O(1) time, O(1) space.
func (q *LinkedQueue[T]) Dequeue() (T, error) {
	if q.head == nil {
		var zero T
		return zero, ErrEmptyQueue
	}
	front := q.head
	q.head = front.next
	front.next = nil
	if q.head == nil {
		q.tail = nil
	}
	q.size--
	return front.value, nil
}

// Peek returns the front without removing it; ErrEmptyQueue if empty. O(1) time, O(1) space.
func (q *LinkedQueue[T]) Peek() (T, error) {
	if q.head == nil {
		var zero T
		return zero, ErrEmptyQueue
	}
	return q.head.value, nil
}

// ToSlice returns the values front to back, empty and non-nil for an empty queue. O(n) time, O(n) space.
func (q *LinkedQueue[T]) ToSlice() []T {
	values := make([]T, 0, q.size)
	for n := q.head; n != nil; n = n.next {
		values = append(values, n.value)
	}
	return values
}

// String renders the queue front first as "10 -> 20 -> 30 -> nil". O(n) time, O(n) space.
func (q *LinkedQueue[T]) String() string {
	var b strings.Builder
	for n := q.head; n != nil; n = n.next {
		fmt.Fprintf(&b, "%v -> ", n.value)
	}
	b.WriteString("nil")
	return b.String()
}

// PrintQueue writes String followed by a newline to standard output. O(n) time, O(n) space.
func (q *LinkedQueue[T]) PrintQueue() {
	fmt.Println(q.String())
}
