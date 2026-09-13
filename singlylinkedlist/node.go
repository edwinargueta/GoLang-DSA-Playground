package singlylinkedlist

// node is one link in the chain: a value and a pointer to its successor.
type node[T comparable] struct {
	value T
	next  *node[T]
}
