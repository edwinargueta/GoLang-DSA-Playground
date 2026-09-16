// Package doublylinkedlist implements a doubly linked list from first principles.
package doublylinkedlist

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrEmptyList is returned by any operation that needs an element and has none.
	ErrEmptyList = errors.New("doublylinkedlist: list is empty")
	// ErrIndexOutOfRange is returned when an index falls outside the list.
	ErrIndexOutOfRange = errors.New("doublylinkedlist: index out of range")
)

// DoublyLinkedList is an ordered sequence of nodes, each holding a pointer to both
// its successor and its predecessor.
//
// Insertion order is preserved and duplicates are kept — this is a sequence, not a
// set. Indices are zero-based and a negative index is always out of range; there is
// no Python-style wraparound.
//
// The list holds head and tail, and every node points backward as well as forward.
// That second pointer is the whole difference from the singly linked list, and it is
// what buys Pop its O(1): the node before the tail is already known rather than found
// by walking. Append, Prepend, PopFront and Pop are all O(1) here.
//
// The structural invariant is mutual: for every node n, n.next.prev == n and
// n.prev.next == n, with head.prev and tail.next both nil. Every mutation must
// restore it before returning. A stale prev pointer is the characteristic bug of this
// structure and it is invisible to a forward traversal, which is why the suite walks
// the chain from both ends and then compares node identity.
//
// Get walks from whichever end is nearer. That halves the constant factor and leaves
// the cost O(n).
//
// T is comparable rather than any because the list searches by value. A zero T is a
// legitimate element, so every lookup reports presence through a separate bool or
// error — never by returning the zero value alone.
//
// The zero DoublyLinkedList is a valid empty list; New is a convenience.
type DoublyLinkedList[T comparable] struct {
	head *node[T]
	tail *node[T]
	size int
}

// New returns an empty list. O(1) time, O(1) space.
func New[T comparable]() *DoublyLinkedList[T] {
	return &DoublyLinkedList[T]{}
}

// Len returns the number of elements. O(1) time, O(1) space.
func (l *DoublyLinkedList[T]) Len() int {
	return l.size
}

// nodeAt returns the node at index, walking from the nearer end; index must be in range.
func (l *DoublyLinkedList[T]) nodeAt(index int) *node[T] {
	if index < l.size/2 {
		n := l.head
		for i := 0; i < index; i++ {
			n = n.next
		}
		return n
	}
	n := l.tail
	for i := l.size - 1; i > index; i-- {
		n = n.prev
	}
	return n
}

// unlink detaches n from the chain, repairs both neighbours, and returns its value.
func (l *DoublyLinkedList[T]) unlink(n *node[T]) T {
	if n.prev == nil {
		l.head = n.next
	} else {
		n.prev.next = n.next
	}
	if n.next == nil {
		l.tail = n.prev
	} else {
		n.next.prev = n.prev
	}
	n.next, n.prev = nil, nil
	l.size--
	return n.value
}

// Append adds value at the tail. O(1) time, O(1) space.
func (l *DoublyLinkedList[T]) Append(value T) {
	n := &node[T]{value: value, prev: l.tail}
	if l.tail == nil {
		l.head = n
	} else {
		l.tail.next = n
	}
	l.tail = n
	l.size++
}

// Prepend adds value at the head. O(1) time, O(1) space.
func (l *DoublyLinkedList[T]) Prepend(value T) {
	n := &node[T]{value: value, next: l.head}
	if l.head == nil {
		l.tail = n
	} else {
		l.head.prev = n
	}
	l.head = n
	l.size++
}

// Get returns the value at index; ErrIndexOutOfRange if index is invalid. O(n) time, O(1) space.
func (l *DoublyLinkedList[T]) Get(index int) (T, error) {
	if index < 0 || index >= l.size {
		var zero T
		return zero, ErrIndexOutOfRange
	}
	return l.nodeAt(index).value, nil
}

// Insert places value at index, shifting the rest right; ErrIndexOutOfRange if index > Len. O(n) time, O(1) space.
func (l *DoublyLinkedList[T]) Insert(index int, value T) error {
	if index < 0 || index > l.size {
		return ErrIndexOutOfRange
	}
	switch index {
	case 0:
		l.Prepend(value)
	case l.size:
		l.Append(value)
	default:
		next := l.nodeAt(index)
		prev := next.prev
		n := &node[T]{value: value, next: next, prev: prev}
		prev.next = n
		next.prev = n
		l.size++
	}
	return nil
}

// PopFront removes and returns the head; ErrEmptyList if empty. O(1) time, O(1) space.
func (l *DoublyLinkedList[T]) PopFront() (T, error) {
	if l.head == nil {
		var zero T
		return zero, ErrEmptyList
	}
	return l.unlink(l.head), nil
}

// Pop removes and returns the tail; ErrEmptyList if empty. O(1) time, O(1) space.
func (l *DoublyLinkedList[T]) Pop() (T, error) {
	if l.tail == nil {
		var zero T
		return zero, ErrEmptyList
	}
	return l.unlink(l.tail), nil
}

// RemoveAt removes and returns the value at index; ErrIndexOutOfRange if index is invalid. O(n) time, O(1) space.
func (l *DoublyLinkedList[T]) RemoveAt(index int) (T, error) {
	if index < 0 || index >= l.size {
		var zero T
		return zero, ErrIndexOutOfRange
	}
	return l.unlink(l.nodeAt(index)), nil
}

// Remove deletes the first node holding value, reporting whether one was found. O(n) time, O(1) space.
func (l *DoublyLinkedList[T]) Remove(value T) bool {
	for n := l.head; n != nil; n = n.next {
		if n.value == value {
			l.unlink(n)
			return true
		}
	}
	return false
}

// IndexOf returns the index of the first node holding value, comma-ok. O(n) time, O(1) space.
func (l *DoublyLinkedList[T]) IndexOf(value T) (int, bool) {
	i := 0
	for n := l.head; n != nil; n = n.next {
		if n.value == value {
			return i, true
		}
		i++
	}
	return 0, false
}

// Contains reports whether any node holds value. O(n) time, O(1) space.
func (l *DoublyLinkedList[T]) Contains(value T) bool {
	for n := l.head; n != nil; n = n.next {
		if n.value == value {
			return true
		}
	}
	return false
}

// Reverse flips the direction of every link in place. O(n) time, O(1) space.
func (l *DoublyLinkedList[T]) Reverse() {
	for n := l.head; n != nil; n = n.prev {
		n.next, n.prev = n.prev, n.next
	}
	l.head, l.tail = l.tail, l.head
}

// HasCycle reports whether the forward chain loops back on itself. O(n) time, O(1) space.
func (l *DoublyLinkedList[T]) HasCycle() bool {
	slow, fast := l.head, l.head
	for fast != nil && fast.next != nil {
		slow = slow.next
		fast = fast.next.next
		if slow == fast {
			return true
		}
	}
	return false
}

// ToSlice returns the values from head to tail, empty and non-nil for an empty list. O(n) time, O(n) space.
func (l *DoublyLinkedList[T]) ToSlice() []T {
	values := make([]T, 0, l.size)
	for n := l.head; n != nil; n = n.next {
		values = append(values, n.value)
	}
	return values
}

// ToSliceReverse returns the values from tail to head, empty and non-nil for an empty list. O(n) time, O(n) space.
func (l *DoublyLinkedList[T]) ToSliceReverse() []T {
	values := make([]T, 0, l.size)
	for n := l.tail; n != nil; n = n.prev {
		values = append(values, n.value)
	}
	return values
}

// String renders the chain as "1 <-> 2 <-> nil". O(n) time, O(n) space.
func (l *DoublyLinkedList[T]) String() string {
	var b strings.Builder
	for n := l.head; n != nil; n = n.next {
		fmt.Fprintf(&b, "%v <-> ", n.value)
	}
	b.WriteString("nil")
	return b.String()
}

// PrintList writes String followed by a newline to standard output. O(n) time, O(n) space.
func (l *DoublyLinkedList[T]) PrintList() {
	fmt.Println(l.String())
}
