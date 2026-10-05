package graph

// edge is one entry in a vertex's adjacency list: the vertex it leads to and its weight.
type edge[T comparable] struct {
	to     T
	weight int
}
