package stack

// node is one cell of the linked stack: a value and a pointer to the cell beneath it.
type node[T any] struct {
	value T
	next  *node[T]
}
