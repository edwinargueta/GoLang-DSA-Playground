// Package graph implements a weighted adjacency-list graph from first principles, with BFS, DFS, cycle detection, topological sort and Dijkstra.
package graph

import "errors"

var (
	// ErrVertexNotFound is returned when a vertex an operation starts from or ends at is not in the graph.
	ErrVertexNotFound = errors.New("graph: vertex not found")
	// ErrNoPath is returned by ShortestPath when the target cannot be reached.
	ErrNoPath = errors.New("graph: no path")
	// ErrNegativeWeight is returned by Dijkstra and ShortestPath when any edge in the graph is negative.
	ErrNegativeWeight = errors.New("graph: negative edge weight")
	// ErrCycle is returned by TopologicalSort when the graph has a cycle and so no order exists.
	ErrCycle = errors.New("graph: graph has a cycle")
	// ErrNotDirected is returned by TopologicalSort on an undirected graph.
	ErrNotDirected = errors.New("graph: graph is not directed")
)

// Graph is a weighted graph stored as adjacency lists: adj maps every vertex to the
// edges leaving it, and vertices records the order the vertices were added in.
//
// A graph is either directed or undirected for its whole life. An undirected edge is
// stored twice, once in each endpoint's list with the same weight, and counted once;
// an undirected self-loop is stored once. There are no parallel edges — at most one
// edge from a vertex to another — so AddEdge on an existing edge replaces its weight
// in place. Weights are ints and may be negative or zero; only Dijkstra and
// ShortestPath refuse negative ones.
//
// Order is part of the contract, because Go's map iteration order is deliberately
// random and a traversal over it would not be repeatable. Vertices come back in the
// order they were added and each adjacency list in the order its edges were added, and
// BFS and DFS visit neighbors in that order — DFS in the order a recursive preorder
// walk produces. Removing a vertex or an edge keeps the order of everything else.
//
// Every vertex has an entry in adj, even with no edges, and every edge leads to a
// vertex in the graph: AddEdge adds a missing endpoint, and RemoveVertex removes the
// edges into a vertex along with the edges out of it.
//
// In an undirected graph an edge and its mirror are one edge, so a-b alone is not a
// cycle; a self-loop is one in either kind of graph. Dijkstra's result holds only the
// vertices source reaches, source itself at distance zero.
//
// It must be made with NewDirected or NewUndirected: the zero value has a nil map and
// no direction.
type Graph[T comparable] struct {
	directed bool
	vertices []T
	adj      map[T][]edge[T]
	edges    int
}

// NewDirected returns an empty directed graph. O(1) time, O(1) space.
func NewDirected[T comparable]() *Graph[T] {
	panic("not implemented")
}

// NewUndirected returns an empty undirected graph. O(1) time, O(1) space.
func NewUndirected[T comparable]() *Graph[T] {
	panic("not implemented")
}

// IsDirected reports whether the graph is directed. O(1) time, O(1) space.
func (g *Graph[T]) IsDirected() bool {
	panic("not implemented")
}

// VertexCount returns the number of vertices. O(1) time, O(1) space.
func (g *Graph[T]) VertexCount() int {
	panic("not implemented")
}

// EdgeCount returns the number of edges, an undirected edge counting once. O(1) time, O(1) space.
func (g *Graph[T]) EdgeCount() int {
	panic("not implemented")
}

// AddVertex adds v with no edges, reporting false if it was already there. O(1) amortized time, O(1) space.
func (g *Graph[T]) AddVertex(v T) bool {
	panic("not implemented")
}

// AddEdge adds an edge from from to to, adding missing endpoints; on an existing edge it replaces the weight and reports false. O(deg(from) + deg(to)) time, O(1) space.
func (g *Graph[T]) AddEdge(from, to T, weight int) bool {
	panic("not implemented")
}

// RemoveEdge removes the edge from from to to, reporting whether it was there. O(deg(from) + deg(to)) time, O(1) space.
func (g *Graph[T]) RemoveEdge(from, to T) bool {
	panic("not implemented")
}

// RemoveVertex removes v and every edge into or out of it, reporting whether it was there. O(V + E) time, O(1) space.
func (g *Graph[T]) RemoveVertex(v T) bool {
	panic("not implemented")
}

// HasVertex reports whether v is in the graph. O(1) time, O(1) space.
func (g *Graph[T]) HasVertex(v T) bool {
	panic("not implemented")
}

// HasEdge reports whether there is an edge from from to to. O(deg(from)) time, O(1) space.
func (g *Graph[T]) HasEdge(from, to T) bool {
	panic("not implemented")
}

// Weight returns the weight of the edge from from to to, comma-ok. O(deg(from)) time, O(1) space.
func (g *Graph[T]) Weight(from, to T) (int, bool) {
	panic("not implemented")
}

// Vertices returns every vertex in the order added, empty and non-nil for an empty graph. O(V) time, O(V) space.
func (g *Graph[T]) Vertices() []T {
	panic("not implemented")
}

// Neighbors returns the vertices v has an edge to, in the order added; ErrVertexNotFound if v is absent. O(deg(v)) time, O(deg(v)) space.
func (g *Graph[T]) Neighbors(v T) ([]T, error) {
	panic("not implemented")
}

// BFS returns the vertices reachable from start in breadth-first order; ErrVertexNotFound if start is absent. O(V + E) time, O(V) space.
func (g *Graph[T]) BFS(start T) ([]T, error) {
	panic("not implemented")
}

// DFS returns the vertices reachable from start in depth-first preorder; ErrVertexNotFound if start is absent. O(V + E) time, O(V) space.
func (g *Graph[T]) DFS(start T) ([]T, error) {
	panic("not implemented")
}

// HasCycle reports whether the graph contains a cycle, a self-loop included. O(V + E) time, O(V) space.
func (g *Graph[T]) HasCycle() bool {
	panic("not implemented")
}

// TopologicalSort returns every vertex ordered so each edge points forward; ErrNotDirected if undirected, ErrCycle if cyclic. O(V + E) time, O(V) space.
func (g *Graph[T]) TopologicalSort() ([]T, error) {
	panic("not implemented")
}

// Dijkstra returns the shortest distance from source to every vertex it reaches; ErrVertexNotFound or ErrNegativeWeight. O((V + E) log V) time, O(V + E) space.
func (g *Graph[T]) Dijkstra(source T) (map[T]int, error) {
	panic("not implemented")
}

// ShortestPath returns a least-weight path from from to to and its total weight; ErrVertexNotFound, ErrNegativeWeight or ErrNoPath. O((V + E) log V) time, O(V + E) space.
func (g *Graph[T]) ShortestPath(from, to T) ([]T, int, error) {
	panic("not implemented")
}

// String renders one line per vertex in the order added, as "a: b(1) c(4)", and "" for an empty graph. O(V + E) time, O(V + E) space.
func (g *Graph[T]) String() string {
	panic("not implemented")
}

// PrintGraph writes String followed by a newline to standard output. O(V + E) time, O(V + E) space.
func (g *Graph[T]) PrintGraph() {
	panic("not implemented")
}
