package binarysearchtree

import "cmp"

// node is one vertex of the tree: a value and pointers to its two subtrees.
type node[T cmp.Ordered] struct {
	value T
	left  *node[T]
	right *node[T]
}
