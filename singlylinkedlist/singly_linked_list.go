// Package singlylinkedlist implements a singly linked list from first principles.
package singlylinkedlist

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrEmptyList is returned by any operation that needs an element and has none.
	ErrEmptyList = errors.New("singlylinkedlist: list is empty")
	// ErrIndexOutOfRange is returned when an index falls outside the list.
	ErrIndexOutOfRange = errors.New("singlylinkedlist: index out of range")
)

// SinglyLinkedList is an ordered sequence of nodes, each holding a pointer only to
// its successor.
//
// Insertion order is preserved and duplicates are kept — this is a sequence, not a
// set. Indices are zero-based and a negative index is always out of range; there is
// no Python-style wraparound.
//
// The list holds head and a size counter; the node holds value and next. There is no
// tail pointer here and no prev pointer on the node, because the second pointer in
// either place is exactly what makes a list doubly linked. Both belong to that
// structure, not this one. Where these comments say "tail" they name a position - the
// last node - never a stored field.
//
// Every operation that touches the end of the chain therefore walks to find it, which
// is why Append is O(n). A tail pointer would buy Append back but still not Pop, since
// removing the last node needs the one before it and no node points backward. That
// asymmetry is the whole reason the doubly linked list exists.
//
// T is comparable rather than any because the list searches by value. A zero T is a
// legitimate element, so every lookup reports presence through a separate bool or
// error — never by returning the zero value alone.
//
// The zero SinglyLinkedList is a valid empty list; New is a convenience.
type SinglyLinkedList[T comparable] struct {
	head *node[T]
	size int
}

// New returns an empty list. O(1) time, O(1) space.
func New[T comparable]() *SinglyLinkedList[T] {
	return &SinglyLinkedList[T]{}
}

// Len returns the number of elements. O(1) time, O(1) space.
func (l *SinglyLinkedList[T]) Len() int {
	return l.size
}

// Append adds value at the tail. O(n) time, O(1) space.
func (l *SinglyLinkedList[T]) Append(value T) {
	n := &node[T]{value: value}
	if l.head == nil {
		l.head = n
		l.size++
		return
	}
	cur := l.head
	for cur.next != nil {
		cur = cur.next
	}
	cur.next = n
	l.size++
}

// Prepend adds value at the head. O(1) time, O(1) space.
func (l *SinglyLinkedList[T]) Prepend(value T) {
	l.head = &node[T]{value: value, next: l.head}
	l.size++
}

// Get returns the value at index; ErrIndexOutOfRange if index is invalid. O(n) time, O(1) space.
func (l *SinglyLinkedList[T]) Get(index int) (T, error) {
	if index < 0 || index >= l.size {
		var zero T
		return zero, ErrIndexOutOfRange
	}
	cur := l.head
	for i := 0; i < index; i++ {
		cur = cur.next
	}
	return cur.value, nil
}

// Insert places value at index, shifting the rest right; ErrIndexOutOfRange if index > Len. O(n) time, O(1) space.
func (l *SinglyLinkedList[T]) Insert(index int, value T) error {
	if index < 0 || index > l.size {
		return ErrIndexOutOfRange
	}
	if index == 0 {
		l.Prepend(value)
		return nil
	}
	// Walk to the node just before the target slot. index-1 is in [0, size-1],
	// so this node always exists; index == size lands us on the last node.
	prev := l.head
	for i := 0; i < index-1; i++ {
		prev = prev.next
	}
	prev.next = &node[T]{value: value, next: prev.next}
	l.size++
	return nil
}

// PopFront removes and returns the head; ErrEmptyList if empty. O(1) time, O(1) space.
func (l *SinglyLinkedList[T]) PopFront() (T, error) {
	if l.head == nil {
		var zero T
		return zero, ErrEmptyList
	}
	n := l.head
	l.head = n.next
	n.next = nil // sever the detached node so it can't pin the rest of the chain
	l.size--
	return n.value, nil
}

// Pop removes and returns the tail; ErrEmptyList if empty. O(n) time, O(1) space.
func (l *SinglyLinkedList[T]) Pop() (T, error) {
	if l.head == nil {
		var zero T
		return zero, ErrEmptyList
	}
	// Single element: the tail is also the head.
	if l.head.next == nil {
		v := l.head.value
		l.head = nil
		l.size--
		return v, nil
	}
	// Stop on the second-to-last node — the one whose next is the tail.
	prev := l.head
	for prev.next.next != nil {
		prev = prev.next
	}
	v := prev.next.value
	prev.next = nil
	l.size--
	return v, nil
}

// RemoveAt removes and returns the value at index; ErrIndexOutOfRange if index is invalid. O(n) time, O(1) space.
func (l *SinglyLinkedList[T]) RemoveAt(index int) (T, error) {
	if index < 0 || index >= l.size {
		var zero T
		return zero, ErrIndexOutOfRange
	}
	if index == 0 {
		return l.PopFront()
	}
	prev := l.head
	for i := 0; i < index-1; i++ {
		prev = prev.next
	}
	removed := prev.next
	prev.next = removed.next
	removed.next = nil
	l.size--
	return removed.value, nil
}

// Remove deletes the first node holding value, reporting whether one was found. O(n) time, O(1) space.
func (l *SinglyLinkedList[T]) Remove(value T) bool {
	if l.head == nil {
		return false
	}
	if l.head.value == value {
		l.head = l.head.next
		l.size--
		return true
	}
	for prev := l.head; prev.next != nil; prev = prev.next {
		if prev.next.value == value {
			prev.next = prev.next.next
			l.size--
			return true
		}
	}
	return false
}

// IndexOf returns the index of the first node holding value, comma-ok. O(n) time, O(1) space.
func (l *SinglyLinkedList[T]) IndexOf(value T) (int, bool) {
	i := 0
	for cur := l.head; cur != nil; cur = cur.next {
		if cur.value == value {
			return i, true
		}
		i++
	}
	return 0, false
}

// Contains reports whether any node holds value. O(n) time, O(1) space.
func (l *SinglyLinkedList[T]) Contains(value T) bool {
	_, ok := l.IndexOf(value)
	return ok
}

// Reverse flips the direction of every link in place. O(n) time, O(1) space.
func (l *SinglyLinkedList[T]) Reverse() {
	var prev *node[T]
	cur := l.head
	for cur != nil {
		next := cur.next // stash before we overwrite it
		cur.next = prev  // point backward
		prev = cur       // advance the trailing pointer
		cur = next       // advance the leading pointer
	}
	l.head = prev
}

// HasCycle reports whether the chain loops back on itself. O(n) time, O(1) space.
func (l *SinglyLinkedList[T]) HasCycle() bool {
	// Floyd's tortoise and hare: if a cycle exists the fast pointer laps the
	// slow one and they meet; otherwise the fast pointer runs off the end.
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

// ToSlice returns the values in order, empty and non-nil for an empty list. O(n) time, O(n) space.
func (l *SinglyLinkedList[T]) ToSlice() []T {
	out := make([]T, 0, l.size) // capacity l.size, length 0 → non-nil even when empty
	for cur := l.head; cur != nil; cur = cur.next {
		out = append(out, cur.value)
	}
	return out
}

// String renders the chain as "1 -> 2 -> nil". O(n) time, O(n) space.
func (l *SinglyLinkedList[T]) String() string {
	var b strings.Builder
	for cur := l.head; cur != nil; cur = cur.next {
		fmt.Fprintf(&b, "%v -> ", cur.value)
	}
	b.WriteString("nil")
	return b.String()
}

// PrintList writes String followed by a newline to standard output. O(n) time, O(n) space.
func (l *SinglyLinkedList[T]) PrintList() {
	fmt.Println(l.String())
}
