package doublylinkedlist

// node is one link in the chain: a value and pointers to both of its neighbours.
type node[T comparable] struct {
	value T
	next  *node[T]
	prev  *node[T]
}
