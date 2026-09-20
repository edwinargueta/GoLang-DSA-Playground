package queue

// node is one cell of the linked queue: a value and a pointer to the cell behind it.
type node[T any] struct {
	value T
	next  *node[T]
}
