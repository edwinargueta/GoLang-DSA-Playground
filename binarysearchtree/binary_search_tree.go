// Package binarysearchtree implements an unbalanced binary search tree from first principles.
package binarysearchtree

import (
	"cmp"
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrEmptyTree is returned by any operation that needs an element and has none.
	ErrEmptyTree = errors.New("binarysearchtree: tree is empty")
)

// BST is a binary search tree: every value in a node's left subtree is strictly less
// than that node's value, and every value in its right subtree is strictly greater.
//
// Duplicates are rejected — this is a set, not a sequence. Insert reports false when
// the value is already present and leaves the tree untouched, the opposite of the
// linked lists in this repo, which keep every copy they are given.
//
// Nothing here rebalances, so insertion order is the whole performance story: random
// input gives a tree of height O(log n), sorted input gives a right spine of height
// n-1 and every operation degrades into the linked list the tree has become. That gap
// is the entire argument for AVL and red-black trees, and leaving it visible is the
// point of writing the unbalanced version first. The costs below are quoted in h, the
// height — read it as log n when the tree is balanced and n when it is not.
//
// Height is measured in edges: an empty tree is -1, a single node is 0. Counting
// nodes instead would make Height and Len agree on a spine, hiding exactly the
// degeneracy above.
//
// The node holds value, left and right, and nothing else. With no parent pointer
// every mutation is written as "rewrite this subtree, return its new root" and the
// caller relinks. Deleting a node with two children copies its in-order successor's
// value up and deletes the successor instead, so node identity is not preserved —
// safe here only because nothing outside the package ever holds a node.
//
// T is cmp.Ordered because the structure orders elements rather than merely comparing
// them for equality. A zero T is a legitimate element, so every lookup reports
// presence through a separate bool or error — never by returning the zero value
// alone. Holding something unordered would mean a less func(a, b T) bool comparator
// on the struct; that is a different design and this is not it.
//
// The zero BST is a valid empty tree; New is a convenience.
type BST[T cmp.Ordered] struct {
	root *node[T]
	size int
}

// New returns an empty tree. O(1) time, O(1) space.
func New[T cmp.Ordered]() *BST[T] {
	return &BST[T]{}
}

// Len returns the number of values in the tree. O(1) time, O(1) space.
func (t *BST[T]) Len() int {
	return t.size
}

// Insert adds value, reporting false if it was already present. O(h) time, O(1) space.
func (t *BST[T]) Insert(value T) bool {
	// Walking a pointer to the link rather than to the node means the empty slot at
	// the end of the descent is written the same way whether it hangs off a parent or
	// is the root itself, so there is no special case for the first insert.
	slot := &t.root
	for *slot != nil {
		switch {
		case value < (*slot).value:
			slot = &(*slot).left
		case value > (*slot).value:
			slot = &(*slot).right
		default:
			return false
		}
	}
	*slot = &node[T]{value: value}
	t.size++
	return true
}

// Contains reports whether value is in the tree. O(h) time, O(1) space.
func (t *BST[T]) Contains(value T) bool {
	for n := t.root; n != nil; {
		switch {
		case value < n.value:
			n = n.left
		case value > n.value:
			n = n.right
		default:
			return true
		}
	}
	return false
}

// Delete removes value, reporting whether it was there to remove. O(h) time, O(h) space.
func (t *BST[T]) Delete(value T) bool {
	root, removed := deleteFrom(t.root, value)
	t.root = root
	if removed {
		t.size--
	}
	return removed
}

// deleteFrom removes value from the subtree at n, returning its new root and whether
// anything was removed.
func deleteFrom[T cmp.Ordered](n *node[T], value T) (*node[T], bool) {
	if n == nil {
		return nil, false
	}

	var removed bool
	switch {
	case value < n.value:
		n.left, removed = deleteFrom(n.left, value)
		return n, removed
	case value > n.value:
		n.right, removed = deleteFrom(n.right, value)
		return n, removed
	}

	// n is the node to remove. With at most one child it is replaced by that child,
	// which is nil when it has none, so both cases collapse into one.
	if n.left == nil {
		return n.right, true
	}
	if n.right == nil {
		return n.left, true
	}

	// With two children the node stays and its value is overwritten by its in-order
	// successor — the smallest value on its right, and the only other value that can
	// sit here without disturbing the ordering. That successor has no left child by
	// construction, so deleting it from the right subtree is one of the cases above.
	succ := n.right
	for succ.left != nil {
		succ = succ.left
	}
	n.value = succ.value
	n.right, _ = deleteFrom(n.right, succ.value)
	return n, true
}

// Min returns the smallest value; ErrEmptyTree if empty. O(h) time, O(1) space.
func (t *BST[T]) Min() (T, error) {
	if t.root == nil {
		var zero T
		return zero, ErrEmptyTree
	}
	n := t.root
	for n.left != nil {
		n = n.left
	}
	return n.value, nil
}

// Max returns the largest value; ErrEmptyTree if empty. O(h) time, O(1) space.
func (t *BST[T]) Max() (T, error) {
	if t.root == nil {
		var zero T
		return zero, ErrEmptyTree
	}
	n := t.root
	for n.right != nil {
		n = n.right
	}
	return n.value, nil
}

// Height returns the longest root-to-leaf path in edges; -1 if empty. O(n) time, O(h) space.
func (t *BST[T]) Height() int {
	return heightOf(t.root)
}

// heightOf returns the height of the subtree at n in edges, -1 for nil so that a leaf works out to 1 + (-1) = 0.
func heightOf[T cmp.Ordered](n *node[T]) int {
	if n == nil {
		return -1
	}
	return 1 + max(heightOf(n.left), heightOf(n.right))
}

// InOrder returns the values left-root-right, which is sorted; empty and non-nil if empty. O(n) time, O(n) space.
func (t *BST[T]) InOrder() []T {
	values := make([]T, 0, t.size)
	var visit func(*node[T])
	visit = func(n *node[T]) {
		if n == nil {
			return
		}
		visit(n.left)
		values = append(values, n.value)
		visit(n.right)
	}
	visit(t.root)
	return values
}

// PreOrder returns the values root-left-right; empty and non-nil if empty. O(n) time, O(n) space.
func (t *BST[T]) PreOrder() []T {
	values := make([]T, 0, t.size)
	var visit func(*node[T])
	visit = func(n *node[T]) {
		if n == nil {
			return
		}
		values = append(values, n.value)
		visit(n.left)
		visit(n.right)
	}
	visit(t.root)
	return values
}

// PostOrder returns the values left-right-root; empty and non-nil if empty. O(n) time, O(n) space.
func (t *BST[T]) PostOrder() []T {
	values := make([]T, 0, t.size)
	var visit func(*node[T])
	visit = func(n *node[T]) {
		if n == nil {
			return
		}
		visit(n.left)
		visit(n.right)
		values = append(values, n.value)
	}
	visit(t.root)
	return values
}

// LevelOrder returns the values breadth-first, top row to bottom; empty and non-nil if empty. O(n) time, O(n) space.
func (t *BST[T]) LevelOrder() []T {
	values := make([]T, 0, t.size)
	if t.root == nil {
		return values
	}
	// Reslicing the front off the queue leaves the consumed cells to the garbage
	// collector rather than shifting the rest down, so the whole walk stays O(n).
	queue := []*node[T]{t.root}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		values = append(values, n.value)
		if n.left != nil {
			queue = append(queue, n.left)
		}
		if n.right != nil {
			queue = append(queue, n.right)
		}
	}
	return values
}

// IsValid reports whether the ordering invariant holds at every node. O(n) time, O(h) space.
func (t *BST[T]) IsValid() bool {
	return within(t.root, nil, nil)
}

// within reports whether every value in the subtree at n lies strictly between low and high, either of which is nil for unbounded.
func within[T cmp.Ordered](n *node[T], low, high *T) bool {
	if n == nil {
		return true
	}
	if low != nil && n.value <= *low {
		return false
	}
	if high != nil && n.value >= *high {
		return false
	}
	// The bound an ancestor imposes has to travel down with the recursion: a value
	// can sit correctly relative to its own parent and still belong in an entirely
	// different subtree, which is what comparing only parent to child misses.
	return within(n.left, low, &n.value) && within(n.right, &n.value, high)
}

// String renders the tree as "50(30(20, 40), 70)", with "." for a missing child. O(n) time, O(n) space.
func (t *BST[T]) String() string {
	if t.root == nil {
		return "()"
	}
	var b strings.Builder
	writeNode(&b, t.root)
	return b.String()
}

// writeNode renders the subtree at n, printing a leaf bare so the parentheses only
// ever mark a branch.
func writeNode[T cmp.Ordered](b *strings.Builder, n *node[T]) {
	if n == nil {
		b.WriteString(".")
		return
	}
	if n.left == nil && n.right == nil {
		fmt.Fprintf(b, "%v", n.value)
		return
	}
	fmt.Fprintf(b, "%v(", n.value)
	writeNode(b, n.left)
	b.WriteString(", ")
	writeNode(b, n.right)
	b.WriteString(")")
}

// PrintTree writes String followed by a newline to standard output. O(n) time, O(n) space.
func (t *BST[T]) PrintTree() {
	fmt.Println(t.String())
}
