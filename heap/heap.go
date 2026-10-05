// Package heap implements a binary heap from first principles, and the k-largest problem on top of it.
package heap

import (
	"cmp"
	"errors"
)

var (
	// ErrEmptyHeap is returned by any operation that needs an element and has none.
	ErrEmptyHeap = errors.New("heap: heap is empty")
)

// Heap is a binary heap stored in a slice: the children of index i sit at 2i+1 and
// 2i+2 and its parent at (i-1)/2, so the tree has no pointers and no gaps, and its
// height is floor(log2 n).
//
// The ordering is a comparator, not cmp.Ordered. less(a, b) reports whether a must
// come out before b, which makes the same type a min-heap, a max-heap, or a priority
// queue over structs ordered by one field — the comparator-function design the other
// doc comments in this repo describe for an ordering the language cannot supply.
// NewMin and NewMax are the cmp.Ordered conveniences.
//
// The invariant is that no element is ordered before its parent: for every i > 0,
// less(items[i], items[(i-1)/2]) is false. That makes items[0] the next element out
// and says nothing about siblings or cousins, so ToSlice and String expose a layout
// that is a heap, not a sorted run. Push restores the invariant by sifting up from the
// new last slot, Pop by moving the last element to the root and sifting down.
//
// Duplicates are kept, and elements that compare equal come out in no particular
// order — a heap is not stable. Pop zeroes the slot it vacates, for the same reason
// the slice-backed stack does. A zero T is a legitimate element, so Pop and Peek
// report emptiness through an error.
//
// It must be made with New, NewMin, NewMax or Heapify: like the hash tables, the
// zero value has no comparator and cannot order anything.
type Heap[T any] struct {
	items []T
	less  func(a, b T) bool
}

// New returns an empty heap ordered by less. O(1) time, O(1) space.
func New[T any](less func(a, b T) bool) *Heap[T] {
	panic("not implemented")
}

// NewMin returns an empty heap that pops the smallest element first. O(1) time, O(1) space.
func NewMin[T cmp.Ordered]() *Heap[T] {
	panic("not implemented")
}

// NewMax returns an empty heap that pops the largest element first. O(1) time, O(1) space.
func NewMax[T cmp.Ordered]() *Heap[T] {
	panic("not implemented")
}

// Heapify returns a heap ordered by less holding a copy of values, leaving values untouched. O(n) time, O(n) space.
func Heapify[T any](values []T, less func(a, b T) bool) *Heap[T] {
	panic("not implemented")
}

// Len returns the number of elements. O(1) time, O(1) space.
func (h *Heap[T]) Len() int {
	panic("not implemented")
}

// IsEmpty reports whether the heap holds no elements. O(1) time, O(1) space.
func (h *Heap[T]) IsEmpty() bool {
	panic("not implemented")
}

// Push adds value and restores the heap order. O(log n) time, O(1) amortized space.
func (h *Heap[T]) Push(value T) {
	panic("not implemented")
}

// Pop removes and returns the root, zeroing the slot it vacates; ErrEmptyHeap if empty. O(log n) time, O(1) space.
func (h *Heap[T]) Pop() (T, error) {
	panic("not implemented")
}

// Peek returns the root without removing it; ErrEmptyHeap if empty. O(1) time, O(1) space.
func (h *Heap[T]) Peek() (T, error) {
	panic("not implemented")
}

// ToSlice returns a copy of the backing array in level order, empty and non-nil for an empty heap. O(n) time, O(n) space.
func (h *Heap[T]) ToSlice() []T {
	panic("not implemented")
}

// String renders the backing array in level order as "[1 3 2]". O(n) time, O(n) space.
func (h *Heap[T]) String() string {
	panic("not implemented")
}

// PrintHeap writes String followed by a newline to standard output. O(n) time, O(n) space.
func (h *Heap[T]) PrintHeap() {
	panic("not implemented")
}
