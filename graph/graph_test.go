// White-box tests for the adjacency-list graph.
//
//	go test ./graph/ -v              # everything in the package
//	go test ./graph/ -run TestBFS -v # one method
//
// Fixtures are written as a vertex list and an edge list, and buildGraph wires them
// into adj by hand the way AddEdge is specified to - appending to the from list, and
// to the to list as well when undirected - so no test depends on AddVertex or
// AddEdge and the methods can be written in any order. The expected state after a
// mutation is written the same way: another buildGraph call, compared field by
// field by assertGraph, adjacency order included, since order is part of the
// contract.
//
// assertGraph also checks the invariants that hold for every graph whatever the
// test: every vertex has an adjacency list and every list belongs to a vertex, no
// edge leads out of the graph, no edge is stored twice, an undirected edge is
// mirrored with the same weight, and the edge counter agrees with the lists. The
// classic RemoveVertex bug - dropping the vertex and its own list but leaving the
// edges that point into it - fails the second of those.
//
// The traversal tests pin exact orders, because neighbor order is part of the
// contract. TopologicalSort, Dijkstra's paths and ShortestPath have more than one
// right answer on most graphs, so those are checked by property instead: the
// order puts every edge forward, the path is made of real edges, and its weight is
// the distance a Bellman-Ford oracle computes independently from adj.
//
// Deliberate exceptions, each noted again above the test itself:
//   - TestGraphShape guards the shape of the structure rather than a method.
//   - TestPrintGraph delegates to String, because printing the rendered graph is
//     the behavior under test.
//   - ExampleGraph exercises NewDirected, AddEdge, BFS and ShortestPath together,
//     because a runnable example is by definition an integration.
package graph

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// e is one edge of a fixture, in the order it was added.
type e[T comparable] struct {
	from, to T
	w        int
}

// es is shorthand for the string-vertex edges almost every fixture uses.
type es = e[string]

// buildGraph returns a graph holding vertices in the given order and edges in the
// given order, wired without calling any method under test. It refuses an edge to a
// vertex not listed and an edge listed twice, since those fixtures break the
// invariants under test.
func buildGraph[T comparable](directed bool, vertices []T, edges ...e[T]) *Graph[T] {
	g := &Graph[T]{directed: directed, vertices: []T{}, adj: map[T][]edge[T]{}}
	for _, v := range vertices {
		if _, dup := g.adj[v]; dup {
			panic(fmt.Sprintf("buildGraph: vertex %v listed twice", v))
		}
		g.vertices = append(g.vertices, v)
		g.adj[v] = []edge[T]{}
	}
	for _, x := range edges {
		if _, ok := g.adj[x.from]; !ok {
			panic(fmt.Sprintf("buildGraph: edge from unlisted vertex %v", x.from))
		}
		if _, ok := g.adj[x.to]; !ok {
			panic(fmt.Sprintf("buildGraph: edge to unlisted vertex %v", x.to))
		}
		for _, have := range g.adj[x.from] {
			if have.to == x.to {
				panic(fmt.Sprintf("buildGraph: edge %v-%v listed twice", x.from, x.to))
			}
		}
		g.adj[x.from] = append(g.adj[x.from], edge[T]{x.to, x.w})
		if !directed && x.from != x.to {
			g.adj[x.to] = append(g.adj[x.to], edge[T]{x.from, x.w})
		}
		g.edges++
	}
	return g
}

// directed and undirected build string graphs from a space-separated vertex list.
func directed(vertices string, edges ...es) *Graph[string] {
	return buildGraph(true, strings.Fields(vertices), edges...)
}

func undirected(vertices string, edges ...es) *Graph[string] {
	return buildGraph(false, strings.Fields(vertices), edges...)
}

// assertGraph checks the graph's invariants and then that it matches want field by
// field, in order. It reads fields only, never a method. Call it after every
// mutation, and after every rejected one too.
func assertGraph[T comparable](t *testing.T, g *Graph[T], want *Graph[T]) {
	t.Helper()

	if g.adj == nil && len(want.vertices) > 0 {
		t.Fatalf("adj is nil, want a map holding %v", want.vertices)
	}
	if g.directed != want.directed {
		t.Errorf("directed = %v, want %v", g.directed, want.directed)
	}

	// Invariants that hold for every graph.
	seen := map[T]bool{}
	for _, v := range g.vertices {
		if seen[v] {
			t.Errorf("vertex %v is listed twice in vertices", v)
		}
		seen[v] = true
		if _, ok := g.adj[v]; !ok {
			t.Errorf("vertex %v has no entry in adj", v)
		}
	}
	entries, loops := 0, 0
	for v, list := range g.adj {
		if !seen[v] {
			t.Errorf("adj has a list for %v, which is not in vertices", v)
		}
		targets := map[T]bool{}
		for _, x := range list {
			entries++
			if _, ok := g.adj[x.to]; !ok {
				t.Errorf("edge %v->%v leads to a vertex not in the graph - a removed vertex left an edge pointing at it", v, x.to)
			}
			if targets[x.to] {
				t.Errorf("edge %v->%v is stored twice", v, x.to)
			}
			targets[x.to] = true
			if x.to == v {
				loops++
				continue
			}
			if !g.directed {
				mirrored := false
				for _, back := range g.adj[x.to] {
					if back.to == v {
						mirrored = true
						if back.weight != x.weight {
							t.Errorf("undirected edge %v-%v has weight %d one way and %d the other", v, x.to, x.weight, back.weight)
						}
					}
				}
				if !mirrored {
					t.Errorf("undirected edge %v->%v has no mirror %v->%v", v, x.to, x.to, v)
				}
			}
		}
	}
	counted := entries
	if !g.directed {
		counted = (entries-loops)/2 + loops
	}
	if g.edges != counted {
		t.Errorf("edges field = %d, but the lists hold %d edges", g.edges, counted)
	}

	// Against the expected graph.
	if !slices.Equal(g.vertices, want.vertices) {
		t.Errorf("vertices = %v, want %v", g.vertices, want.vertices)
	}
	for _, v := range want.vertices {
		if got, w := g.adj[v], want.adj[v]; !slices.Equal(got, w) {
			t.Errorf("adj[%v] = %v, want %v", v, got, w)
		}
	}
	if g.edges != want.edges {
		t.Errorf("edges field = %d, want %d", g.edges, want.edges)
	}
}

// distances is the Bellman-Ford oracle: shortest distances from source read
// straight from adj, independent of anything under test.
func distances[T comparable](g *Graph[T], source T) map[T]int {
	dist := map[T]int{source: 0}
	for range g.vertices {
		for _, v := range g.vertices {
			d, ok := dist[v]
			if !ok {
				continue
			}
			for _, x := range g.adj[v] {
				if cur, seen := dist[x.to]; !seen || d+x.weight < cur {
					dist[x.to] = d + x.weight
				}
			}
		}
	}
	return dist
}

// weightOf returns the weight of the edge from from to to by reading adj.
func weightOf[T comparable](g *Graph[T], from, to T) (int, bool) {
	for _, x := range g.adj[from] {
		if x.to == to {
			return x.weight, true
		}
	}
	return 0, false
}

// assertTopological checks that order holds every vertex exactly once and that
// every edge points forward in it.
func assertTopological[T comparable](t *testing.T, g *Graph[T], order []T) {
	t.Helper()
	position := map[T]int{}
	for i, v := range order {
		if _, dup := position[v]; dup {
			t.Errorf("order %v lists %v twice", order, v)
		}
		position[v] = i
	}
	if len(position) != len(g.vertices) {
		t.Errorf("order %v holds %d vertices, want all %d", order, len(position), len(g.vertices))
	}
	for v, list := range g.adj {
		for _, x := range list {
			if position[v] >= position[x.to] {
				t.Errorf("order %v puts %v before %v, against the edge %v->%v", order, x.to, v, v, x.to)
			}
		}
	}
}

// captureStdout redirects os.Stdout for the duration of f and returns what was
// written.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	f()

	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("reading captured stdout failed: %v", err)
	}
	return buf.String()
}

// TestGraphShape pins the shape of the structure rather than any one method.
// Adjacency lists in a map, with a separate slice for order because the map has
// none, and a counter so EdgeCount is O(1).
func TestGraphShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  reflect.Type
		want []string
	}{
		{"graph", reflect.TypeOf(Graph[int]{}), []string{"directed", "vertices", "adj", "edges"}},
		{"edge", reflect.TypeOf(edge[int]{}), []string{"to", "weight"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, f := range reflect.VisibleFields(tc.typ) {
				got = append(got, f.Name)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("%s fields = %v, want %v", tc.typ.Name(), got, tc.want)
			}
		})
	}
}

// TestGraphEmpty is the case most implementations get wrong, so every method gets
// a subtest against a freshly built empty graph.
func TestGraphEmpty(t *testing.T) {
	t.Run("VertexCount", func(t *testing.T) {
		if got := directed("").VertexCount(); got != 0 {
			t.Errorf("VertexCount() = %d, want 0", got)
		}
	})
	t.Run("EdgeCount", func(t *testing.T) {
		if got := directed("").EdgeCount(); got != 0 {
			t.Errorf("EdgeCount() = %d, want 0", got)
		}
	})
	t.Run("AddVertex", func(t *testing.T) {
		g := directed("")
		if got := g.AddVertex("a"); !got {
			t.Errorf("AddVertex(a) = false, want true")
		}
		assertGraph(t, g, directed("a"))
	})
	t.Run("AddEdge", func(t *testing.T) {
		g := directed("")
		if got := g.AddEdge("a", "b", 1); !got {
			t.Errorf("AddEdge(a, b, 1) = false, want true")
		}
		assertGraph(t, g, directed("a b", es{"a", "b", 1}))
	})
	t.Run("RemoveEdge", func(t *testing.T) {
		g := directed("")
		if got := g.RemoveEdge("a", "b"); got {
			t.Errorf("RemoveEdge(a, b) = true, want false")
		}
		assertGraph(t, g, directed(""))
	})
	t.Run("RemoveVertex", func(t *testing.T) {
		g := directed("")
		if got := g.RemoveVertex("a"); got {
			t.Errorf("RemoveVertex(a) = true, want false")
		}
		assertGraph(t, g, directed(""))
	})
	t.Run("HasVertex", func(t *testing.T) {
		if directed("").HasVertex("a") {
			t.Error("HasVertex(a) = true, want false")
		}
	})
	t.Run("HasEdge", func(t *testing.T) {
		if directed("").HasEdge("a", "b") {
			t.Error("HasEdge(a, b) = true, want false")
		}
	})
	t.Run("Weight", func(t *testing.T) {
		if w, ok := directed("").Weight("a", "b"); ok {
			t.Errorf("Weight(a, b) = %d, true, want ok false", w)
		}
	})
	t.Run("Vertices", func(t *testing.T) {
		got := directed("").Vertices()
		if got == nil || len(got) != 0 {
			t.Errorf("Vertices() = %#v, want an empty non-nil slice", got)
		}
	})
	t.Run("Neighbors", func(t *testing.T) {
		if _, err := directed("").Neighbors("a"); !errors.Is(err, ErrVertexNotFound) {
			t.Errorf("Neighbors(a) error = %v, want ErrVertexNotFound", err)
		}
	})
	t.Run("BFS", func(t *testing.T) {
		if _, err := directed("").BFS("a"); !errors.Is(err, ErrVertexNotFound) {
			t.Errorf("BFS(a) error = %v, want ErrVertexNotFound", err)
		}
	})
	t.Run("DFS", func(t *testing.T) {
		if _, err := directed("").DFS("a"); !errors.Is(err, ErrVertexNotFound) {
			t.Errorf("DFS(a) error = %v, want ErrVertexNotFound", err)
		}
	})
	t.Run("HasCycle", func(t *testing.T) {
		if directed("").HasCycle() || undirected("").HasCycle() {
			t.Error("HasCycle() = true, want false")
		}
	})
	t.Run("TopologicalSort", func(t *testing.T) {
		got, err := directed("").TopologicalSort()
		if err != nil {
			t.Fatalf("TopologicalSort() error = %v, want nil", err)
		}
		if got == nil || len(got) != 0 {
			t.Errorf("TopologicalSort() = %#v, want an empty non-nil slice", got)
		}
	})
	t.Run("Dijkstra", func(t *testing.T) {
		if _, err := directed("").Dijkstra("a"); !errors.Is(err, ErrVertexNotFound) {
			t.Errorf("Dijkstra(a) error = %v, want ErrVertexNotFound", err)
		}
	})
	t.Run("ShortestPath", func(t *testing.T) {
		if _, _, err := directed("").ShortestPath("a", "b"); !errors.Is(err, ErrVertexNotFound) {
			t.Errorf("ShortestPath(a, b) error = %v, want ErrVertexNotFound", err)
		}
	})
	t.Run("String", func(t *testing.T) {
		if got := directed("").String(); got != "" {
			t.Errorf("String() = %q, want %q", got, "")
		}
	})
}

func TestNewDirected(t *testing.T) {
	g := NewDirected[string]()
	if g == nil {
		t.Fatal("NewDirected() = nil, want a graph")
	}
	assertGraph(t, g, directed(""))
}

func TestNewUndirected(t *testing.T) {
	g := NewUndirected[int]()
	if g == nil {
		t.Fatal("NewUndirected() = nil, want a graph")
	}
	assertGraph(t, g, buildGraph[int](false, nil))
}

func TestIsDirected(t *testing.T) {
	if !directed("a").IsDirected() {
		t.Error("IsDirected() = false on a directed graph, want true")
	}
	if undirected("a").IsDirected() {
		t.Error("IsDirected() = true on an undirected graph, want false")
	}
}

func TestVertexCount(t *testing.T) {
	for _, tc := range []struct {
		name string
		g    *Graph[string]
		want int
	}{
		{"one", directed("a"), 1},
		{"isolated vertices count", directed("a b c"), 3},
		{"with edges", undirected("a b c", es{"a", "b", 1}), 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.g.VertexCount(); got != tc.want {
				t.Errorf("VertexCount() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestEdgeCount(t *testing.T) {
	for _, tc := range []struct {
		name string
		g    *Graph[string]
		want int
	}{
		{"no edges", directed("a b"), 0},
		{"directed", directed("a b c", es{"a", "b", 1}, es{"b", "a", 1}, es{"b", "c", 1}), 3},
		{"undirected counts once", undirected("a b c", es{"a", "b", 1}, es{"b", "c", 1}), 2},
		{"self-loop", undirected("a", es{"a", "a", 1}), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.g.EdgeCount(); got != tc.want {
				t.Errorf("EdgeCount() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestAddVertex(t *testing.T) {
	t.Run("appends in order", func(t *testing.T) {
		g := directed("a b")
		if got := g.AddVertex("c"); !got {
			t.Errorf("AddVertex(c) = false, want true")
		}
		assertGraph(t, g, directed("a b c"))
	})
	t.Run("existing vertex keeps its edges", func(t *testing.T) {
		g := directed("a b", es{"a", "b", 5})
		if got := g.AddVertex("a"); got {
			t.Errorf("AddVertex(a) = true, want false - a is already there")
		}
		assertGraph(t, g, directed("a b", es{"a", "b", 5}))
	})
	t.Run("zero value vertex", func(t *testing.T) {
		g := buildGraph(true, []int{1})
		if got := g.AddVertex(0); !got {
			t.Errorf("AddVertex(0) = false, want true - 0 is a vertex like any other")
		}
		assertGraph(t, g, buildGraph(true, []int{1, 0}))
	})
}

func TestAddEdge(t *testing.T) {
	for _, tc := range []struct {
		name     string
		start    *Graph[string]
		from, to string
		weight   int
		want     bool
		end      *Graph[string]
	}{
		{"directed", directed("a b"), "a", "b", 3, true, directed("a b", es{"a", "b", 3})},
		{"undirected mirrors", undirected("a b"), "a", "b", 3, true, undirected("a b", es{"a", "b", 3})},
		{"appends after existing edges", directed("a b c", es{"a", "b", 1}), "a", "c", 2, true, directed("a b c", es{"a", "b", 1}, es{"a", "c", 2})},
		{"adds a missing from", directed("b"), "a", "b", 1, true, directed("b a", es{"a", "b", 1})},
		{"adds a missing to", directed("a"), "a", "b", 1, true, directed("a b", es{"a", "b", 1})},
		{"adds both, from first", directed("x"), "a", "b", 1, true, directed("x a b", es{"a", "b", 1})},
		{"reverse is a new edge when directed", directed("a b", es{"a", "b", 1}), "b", "a", 2, true, directed("a b", es{"a", "b", 1}, es{"b", "a", 2})},
		{"existing edge replaces its weight in place", directed("a b c", es{"a", "b", 1}, es{"a", "c", 2}), "a", "b", 9, false, directed("a b c", es{"a", "b", 9}, es{"a", "c", 2})},
		{"undirected reverse is the same edge", undirected("a b", es{"a", "b", 1}), "b", "a", 7, false, undirected("a b", es{"a", "b", 7})},
		{"self-loop directed", directed("a"), "a", "a", 1, true, directed("a", es{"a", "a", 1})},
		{"self-loop undirected is stored once", undirected("a"), "a", "a", 1, true, undirected("a", es{"a", "a", 1})},
		{"zero weight", directed("a b"), "a", "b", 0, true, directed("a b", es{"a", "b", 0})},
		{"negative weight", directed("a b"), "a", "b", -4, true, directed("a b", es{"a", "b", -4})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.start.AddEdge(tc.from, tc.to, tc.weight); got != tc.want {
				t.Errorf("AddEdge(%s, %s, %d) = %v, want %v", tc.from, tc.to, tc.weight, got, tc.want)
			}
			assertGraph(t, tc.start, tc.end)
		})
	}
}

func TestRemoveEdge(t *testing.T) {
	for _, tc := range []struct {
		name     string
		start    *Graph[string]
		from, to string
		want     bool
		end      *Graph[string]
	}{
		{"directed", directed("a b", es{"a", "b", 1}), "a", "b", true, directed("a b")},
		{"directed leaves the reverse", directed("a b", es{"a", "b", 1}, es{"b", "a", 2}), "a", "b", true, directed("a b", es{"b", "a", 2})},
		{"directed reverse is not the edge", directed("a b", es{"a", "b", 1}), "b", "a", false, directed("a b", es{"a", "b", 1})},
		{"undirected removes both halves", undirected("a b", es{"a", "b", 1}), "a", "b", true, undirected("a b")},
		{"undirected from the other end", undirected("a b", es{"a", "b", 1}), "b", "a", true, undirected("a b")},
		{"keeps the order of the rest", directed("a b c d", es{"a", "b", 1}, es{"a", "c", 2}, es{"a", "d", 3}), "a", "c", true, directed("a b c d", es{"a", "b", 1}, es{"a", "d", 3})},
		{"first of several", directed("a b c", es{"a", "b", 1}, es{"a", "c", 2}), "a", "b", true, directed("a b c", es{"a", "c", 2})},
		{"last of several", directed("a b c", es{"a", "b", 1}, es{"a", "c", 2}), "a", "c", true, directed("a b c", es{"a", "b", 1})},
		{"undirected in a busy graph", undirected("a b c", es{"a", "b", 1}, es{"b", "c", 2}, es{"c", "a", 3}), "b", "c", true, undirected("a b c", es{"a", "b", 1}, es{"c", "a", 3})},
		{"self-loop directed", directed("a", es{"a", "a", 1}), "a", "a", true, directed("a")},
		{"self-loop undirected", undirected("a b", es{"a", "a", 1}, es{"a", "b", 2}), "a", "a", true, undirected("a b", es{"a", "b", 2})},
		{"absent edge", directed("a b c", es{"a", "b", 1}), "a", "c", false, directed("a b c", es{"a", "b", 1})},
		{"absent from", directed("a"), "x", "a", false, directed("a")},
		{"absent to", directed("a"), "a", "x", false, directed("a")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.start.RemoveEdge(tc.from, tc.to); got != tc.want {
				t.Errorf("RemoveEdge(%s, %s) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
			assertGraph(t, tc.start, tc.end)
		})
	}
}

func TestRemoveVertex(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start *Graph[string]
		v     string
		want  bool
		end   *Graph[string]
	}{
		{"only vertex", directed("a"), "a", true, directed("")},
		{"isolated", directed("a b c"), "b", true, directed("a c")},
		{"first", directed("a b c"), "a", true, directed("b c")},
		{"last", directed("a b c"), "c", true, directed("a b")},
		{"with edges out", directed("a b c", es{"a", "b", 1}, es{"a", "c", 2}), "a", true, directed("b c")},
		{"with edges in", directed("a b c", es{"a", "b", 1}, es{"c", "b", 2}, es{"a", "c", 3}), "b", true, directed("a c", es{"a", "c", 3})},
		{"in and out", directed("a b c", es{"a", "b", 1}, es{"b", "c", 2}, es{"c", "a", 3}), "b", true, directed("a c", es{"c", "a", 3})},
		{"undirected", undirected("a b c d", es{"a", "b", 1}, es{"b", "c", 2}, es{"c", "d", 3}, es{"a", "d", 4}), "b", true, undirected("a c d", es{"c", "d", 3}, es{"a", "d", 4})},
		{"with a self-loop", directed("a b", es{"a", "a", 1}, es{"a", "b", 2}), "a", true, directed("b")},
		{"keeps the order of other lists", directed("a b c d", es{"a", "b", 1}, es{"a", "c", 2}, es{"a", "d", 3}), "c", true, directed("a b d", es{"a", "b", 1}, es{"a", "d", 3})},
		{"absent", directed("a b", es{"a", "b", 1}), "x", false, directed("a b", es{"a", "b", 1})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.start.RemoveVertex(tc.v); got != tc.want {
				t.Errorf("RemoveVertex(%s) = %v, want %v", tc.v, got, tc.want)
			}
			assertGraph(t, tc.start, tc.end)
		})
	}

	t.Run("zero value vertex", func(t *testing.T) {
		g := buildGraph(true, []int{0, 1}, e[int]{1, 0, 5})
		if got := g.RemoveVertex(0); !got {
			t.Errorf("RemoveVertex(0) = false, want true")
		}
		assertGraph(t, g, buildGraph(true, []int{1}))
	})
}

func TestHasVertex(t *testing.T) {
	g := buildGraph(true, []int{0, 1, 2})
	for _, tc := range []struct {
		v    int
		want bool
	}{
		{0, true},
		{2, true},
		{3, false},
		{-1, false},
	} {
		if got := g.HasVertex(tc.v); got != tc.want {
			t.Errorf("HasVertex(%d) = %v, want %v", tc.v, got, tc.want)
		}
	}
}

func TestHasEdge(t *testing.T) {
	for _, tc := range []struct {
		name     string
		g        *Graph[string]
		from, to string
		want     bool
	}{
		{"directed", directed("a b", es{"a", "b", 1}), "a", "b", true},
		{"directed reverse", directed("a b", es{"a", "b", 1}), "b", "a", false},
		{"undirected reverse", undirected("a b", es{"a", "b", 1}), "b", "a", true},
		{"zero weight still an edge", directed("a b", es{"a", "b", 0}), "a", "b", true},
		{"self-loop", directed("a", es{"a", "a", 1}), "a", "a", true},
		{"no self-loop", directed("a"), "a", "a", false},
		{"not adjacent", directed("a b c", es{"a", "b", 1}), "a", "c", false},
		{"absent vertex", directed("a"), "a", "x", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.g.HasEdge(tc.from, tc.to); got != tc.want {
				t.Errorf("HasEdge(%s, %s) = %v, want %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestWeight(t *testing.T) {
	for _, tc := range []struct {
		name     string
		g        *Graph[string]
		from, to string
		want     int
		ok       bool
	}{
		{"directed", directed("a b", es{"a", "b", 7}), "a", "b", 7, true},
		{"second in the list", directed("a b c", es{"a", "b", 7}, es{"a", "c", 9}), "a", "c", 9, true},
		{"undirected reverse", undirected("a b", es{"a", "b", 7}), "b", "a", 7, true},
		{"zero weight is present", directed("a b", es{"a", "b", 0}), "a", "b", 0, true},
		{"negative", directed("a b", es{"a", "b", -2}), "a", "b", -2, true},
		{"directed reverse is absent", directed("a b", es{"a", "b", 7}), "b", "a", 0, false},
		{"absent vertex", directed("a"), "x", "a", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := tc.g.Weight(tc.from, tc.to)
			if ok != tc.ok {
				t.Fatalf("Weight(%s, %s) ok = %v, want %v", tc.from, tc.to, ok, tc.ok)
			}
			if got != tc.want {
				t.Errorf("Weight(%s, %s) = %d, want %d", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestVertices(t *testing.T) {
	t.Run("in the order added", func(t *testing.T) {
		g := directed("c a b", es{"b", "a", 1})
		if got, want := g.Vertices(), []string{"c", "a", "b"}; !slices.Equal(got, want) {
			t.Errorf("Vertices() = %v, want %v", got, want)
		}
	})
	t.Run("returns a copy", func(t *testing.T) {
		g := directed("a b")
		got := g.Vertices()
		got[0] = "z"
		assertGraph(t, g, directed("a b"))
	})
}

func TestNeighbors(t *testing.T) {
	for _, tc := range []struct {
		name string
		g    *Graph[string]
		v    string
		want []string
	}{
		{"isolated", directed("a b"), "a", []string{}},
		{"in the order added", directed("a b c d", es{"a", "d", 1}, es{"a", "b", 1}, es{"a", "c", 1}), "a", []string{"d", "b", "c"}},
		{"directed has no back edges", directed("a b", es{"a", "b", 1}), "b", []string{}},
		{"undirected sees both ways", undirected("a b c", es{"a", "b", 1}, es{"c", "b", 1}), "b", []string{"a", "c"}},
		{"self-loop", directed("a b", es{"a", "a", 1}, es{"a", "b", 1}), "a", []string{"a", "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.g.Neighbors(tc.v)
			if err != nil {
				t.Fatalf("Neighbors(%s) error = %v, want nil", tc.v, err)
			}
			if got == nil || !slices.Equal(got, tc.want) {
				t.Errorf("Neighbors(%s) = %#v, want %#v", tc.v, got, tc.want)
			}
		})
	}

	t.Run("absent", func(t *testing.T) {
		g := directed("a")
		if _, err := g.Neighbors("x"); !errors.Is(err, ErrVertexNotFound) {
			t.Errorf("Neighbors(x) error = %v, want ErrVertexNotFound", err)
		}
	})
	t.Run("returns a copy", func(t *testing.T) {
		g := directed("a b", es{"a", "b", 1})
		got, _ := g.Neighbors("a")
		if len(got) > 0 {
			got[0] = "z"
		}
		assertGraph(t, g, directed("a b", es{"a", "b", 1}))
	})
}

// traversalCases are shared by BFS and DFS: the same graphs, with the order each
// traversal must produce. The two columns differ exactly where breadth-first and
// depth-first differ.
var traversalCases = []struct {
	name  string
	g     *Graph[string]
	start string
	bfs   []string
	dfs   []string
}{
	{"single vertex", directed("a"), "a", []string{"a"}, []string{"a"}},
	{"chain", directed("a b c", es{"a", "b", 1}, es{"b", "c", 1}), "a", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
	{"branches", directed("a b c d", es{"a", "b", 1}, es{"a", "c", 1}, es{"b", "d", 1}), "a", []string{"a", "b", "c", "d"}, []string{"a", "b", "d", "c"}},
	{"neighbor order decides", directed("a b c d", es{"a", "c", 1}, es{"a", "b", 1}, es{"c", "d", 1}), "a", []string{"a", "c", "b", "d"}, []string{"a", "c", "d", "b"}},
	{"diamond visits once", directed("a b c d", es{"a", "b", 1}, es{"a", "c", 1}, es{"b", "d", 1}, es{"c", "d", 1}), "a", []string{"a", "b", "c", "d"}, []string{"a", "b", "d", "c"}},
	{"cycle terminates", directed("a b c", es{"a", "b", 1}, es{"b", "c", 1}, es{"c", "a", 1}), "a", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
	{"self-loop", directed("a b", es{"a", "a", 1}, es{"a", "b", 1}), "a", []string{"a", "b"}, []string{"a", "b"}},
	{"only what is reachable", directed("a b c d", es{"a", "b", 1}, es{"c", "d", 1}), "a", []string{"a", "b"}, []string{"a", "b"}},
	{"directed edges go one way", directed("a b c", es{"a", "b", 1}, es{"b", "c", 1}), "b", []string{"b", "c"}, []string{"b", "c"}},
	{"undirected edges go both ways", undirected("a b c", es{"a", "b", 1}, es{"b", "c", 1}), "b", []string{"b", "a", "c"}, []string{"b", "a", "c"}},
	{"from a later vertex", directed("a b c", es{"c", "a", 1}, es{"a", "b", 1}), "c", []string{"c", "a", "b"}, []string{"c", "a", "b"}},
	// A stack-based DFS that marks a vertex when it is pushed rather than when it is
	// visited reaches c from a's list before d gets to it, and comes out a b d f c e.
	{"preorder, not mark-on-push", directed("a b c d e f", es{"a", "b", 1}, es{"a", "c", 1}, es{"b", "d", 1}, es{"b", "f", 1}, es{"d", "c", 1}, es{"c", "e", 1}), "a",
		[]string{"a", "b", "c", "d", "f", "e"}, []string{"a", "b", "d", "c", "e", "f"}},
	{"levels", undirected("r a b c d e", es{"r", "a", 1}, es{"r", "b", 1}, es{"a", "c", 1}, es{"a", "d", 1}, es{"b", "e", 1}), "r",
		[]string{"r", "a", "b", "c", "d", "e"}, []string{"r", "a", "c", "d", "b", "e"}},
}

func TestBFS(t *testing.T) {
	for _, tc := range traversalCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.g.BFS(tc.start)
			if err != nil {
				t.Fatalf("BFS(%s) error = %v, want nil", tc.start, err)
			}
			if !slices.Equal(got, tc.bfs) {
				t.Errorf("BFS(%s) = %v, want %v", tc.start, got, tc.bfs)
			}
		})
	}
	t.Run("absent start", func(t *testing.T) {
		if _, err := directed("a").BFS("x"); !errors.Is(err, ErrVertexNotFound) {
			t.Errorf("BFS(x) error = %v, want ErrVertexNotFound", err)
		}
	})
	t.Run("zero value vertex", func(t *testing.T) {
		g := buildGraph(true, []int{0, 1}, e[int]{0, 1, 1})
		if got, err := g.BFS(0); err != nil || !slices.Equal(got, []int{0, 1}) {
			t.Errorf("BFS(0) = %v, %v, want [0 1], nil", got, err)
		}
	})
}

func TestDFS(t *testing.T) {
	for _, tc := range traversalCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.g.DFS(tc.start)
			if err != nil {
				t.Fatalf("DFS(%s) error = %v, want nil", tc.start, err)
			}
			if !slices.Equal(got, tc.dfs) {
				t.Errorf("DFS(%s) = %v, want %v", tc.start, got, tc.dfs)
			}
		})
	}
	t.Run("absent start", func(t *testing.T) {
		if _, err := directed("a").DFS("x"); !errors.Is(err, ErrVertexNotFound) {
			t.Errorf("DFS(x) error = %v, want ErrVertexNotFound", err)
		}
	})
	t.Run("zero value vertex", func(t *testing.T) {
		g := buildGraph(true, []int{0, 1}, e[int]{0, 1, 1})
		if got, err := g.DFS(0); err != nil || !slices.Equal(got, []int{0, 1}) {
			t.Errorf("DFS(0) = %v, %v, want [0 1], nil", got, err)
		}
	})
}

func TestHasCycle(t *testing.T) {
	for _, tc := range []struct {
		name string
		g    *Graph[string]
		want bool
	}{
		{"single vertex", directed("a"), false},
		{"directed chain", directed("a b c", es{"a", "b", 1}, es{"b", "c", 1}), false},
		{"directed two-cycle", directed("a b", es{"a", "b", 1}, es{"b", "a", 1}), true},
		{"directed triangle", directed("a b c", es{"a", "b", 1}, es{"b", "c", 1}, es{"c", "a", 1}), true},
		{"directed self-loop", directed("a", es{"a", "a", 1}), true},
		// Reaching d a second time is not a cycle. A directed check that only tracks
		// visited, not which vertices are still on the current path, says it is.
		{"directed diamond", directed("a b c d", es{"a", "b", 1}, es{"a", "c", 1}, es{"b", "d", 1}, es{"c", "d", 1}), false},
		{"directed cross edge to a finished branch", directed("a b c", es{"a", "b", 1}, es{"c", "b", 1}), false},
		{"directed cycle unreachable from the first vertex", directed("a b c d", es{"a", "b", 1}, es{"c", "d", 1}, es{"d", "c", 1}), true},
		{"undirected single edge", undirected("a b", es{"a", "b", 1}), false},
		{"undirected tree", undirected("a b c d", es{"a", "b", 1}, es{"a", "c", 1}, es{"c", "d", 1}), false},
		{"undirected triangle", undirected("a b c", es{"a", "b", 1}, es{"b", "c", 1}, es{"c", "a", 1}), true},
		{"undirected self-loop", undirected("a", es{"a", "a", 1}), true},
		{"undirected cycle in a second component", undirected("a b c d e", es{"a", "b", 1}, es{"c", "d", 1}, es{"d", "e", 1}, es{"e", "c", 1}), true},
		{"undirected forest", undirected("a b c d", es{"a", "b", 1}, es{"c", "d", 1}), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.g.HasCycle(); got != tc.want {
				t.Errorf("HasCycle() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTopologicalSort(t *testing.T) {
	for _, tc := range []struct {
		name string
		g    *Graph[string]
	}{
		{"single vertex", directed("a")},
		{"isolated vertices", directed("a b c")},
		{"chain", directed("a b c", es{"a", "b", 1}, es{"b", "c", 1})},
		{"chain added backwards", directed("c b a", es{"b", "c", 1}, es{"a", "b", 1})},
		{"diamond", directed("a b c d", es{"a", "b", 1}, es{"a", "c", 1}, es{"b", "d", 1}, es{"c", "d", 1})},
		{"two components", directed("a b c d", es{"b", "a", 1}, es{"d", "c", 1})},
		{"course prerequisites", directed("calc1 calc2 linalg ml stats",
			es{"calc1", "calc2", 1}, es{"calc2", "ml", 1}, es{"linalg", "ml", 1}, es{"stats", "ml", 1}, es{"calc1", "linalg", 1})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.g.TopologicalSort()
			if err != nil {
				t.Fatalf("TopologicalSort() error = %v, want nil", err)
			}
			assertTopological(t, tc.g, got)
		})
	}

	for _, tc := range []struct {
		name string
		g    *Graph[string]
		want error
	}{
		{"cycle", directed("a b c", es{"a", "b", 1}, es{"b", "c", 1}, es{"c", "a", 1}), ErrCycle},
		{"self-loop", directed("a b", es{"a", "b", 1}, es{"b", "b", 1}), ErrCycle},
		{"cycle off to one side", directed("a b c d", es{"a", "b", 1}, es{"c", "d", 1}, es{"d", "c", 1}), ErrCycle},
		{"undirected", undirected("a b", es{"a", "b", 1}), ErrNotDirected},
		{"undirected without edges", undirected("a"), ErrNotDirected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.g.TopologicalSort(); !errors.Is(err, tc.want) {
				t.Errorf("TopologicalSort() error = %v, want %v", err, tc.want)
			}
		})
	}
}

// shortestCases are shared by Dijkstra and ShortestPath. Every case is checked
// against the Bellman-Ford oracle rather than a hand-written answer.
var shortestCases = []struct {
	name   string
	g      *Graph[string]
	source string
}{
	{"single vertex", directed("a"), "a"},
	{"chain", directed("a b c", es{"a", "b", 2}, es{"b", "c", 3}), "a"},
	{"longer path is cheaper", directed("a b c", es{"a", "b", 10}, es{"a", "c", 1}, es{"c", "b", 2}), "a"},
	// b is reached first at 9 and must be improved to 3 after it is already known.
	{"a reached vertex improves later", directed("a b c d", es{"a", "b", 9}, es{"a", "c", 1}, es{"c", "d", 1}, es{"d", "b", 1}), "a"},
	{"improvement propagates", directed("a b c d", es{"a", "b", 5}, es{"b", "d", 1}, es{"a", "c", 1}, es{"c", "b", 1}), "a"},
	{"zero weights", directed("a b c", es{"a", "b", 0}, es{"b", "c", 0}), "a"},
	{"unreachable vertices", directed("a b c d", es{"a", "b", 1}, es{"c", "d", 1}), "a"},
	{"self-loop on the source", directed("a b", es{"a", "a", 4}, es{"a", "b", 1}), "a"},
	{"cycle", directed("a b c", es{"a", "b", 1}, es{"b", "c", 1}, es{"c", "a", 1}), "b"},
	{"undirected", undirected("a b c d", es{"a", "b", 4}, es{"b", "c", 1}, es{"c", "d", 1}, es{"a", "d", 1}), "a"},
	{"grid", undirected("a b c d e f g h i",
		es{"a", "b", 1}, es{"b", "c", 7}, es{"d", "e", 2}, es{"e", "f", 1}, es{"g", "h", 3}, es{"h", "i", 1},
		es{"a", "d", 4}, es{"d", "g", 1}, es{"b", "e", 2}, es{"e", "h", 6}, es{"c", "f", 1}, es{"f", "i", 2}), "a"},
}

func TestDijkstra(t *testing.T) {
	for _, tc := range shortestCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.g.Dijkstra(tc.source)
			if err != nil {
				t.Fatalf("Dijkstra(%s) error = %v, want nil", tc.source, err)
			}
			if want := distances(tc.g, tc.source); !maps.Equal(got, want) {
				t.Errorf("Dijkstra(%s) = %v, want %v", tc.source, got, want)
			}
		})
	}

	t.Run("absent source", func(t *testing.T) {
		if _, err := directed("a").Dijkstra("x"); !errors.Is(err, ErrVertexNotFound) {
			t.Errorf("Dijkstra(x) error = %v, want ErrVertexNotFound", err)
		}
	})
	t.Run("negative edge", func(t *testing.T) {
		g := directed("a b", es{"a", "b", -1})
		if _, err := g.Dijkstra("a"); !errors.Is(err, ErrNegativeWeight) {
			t.Errorf("Dijkstra(a) error = %v, want ErrNegativeWeight", err)
		}
	})
	t.Run("negative edge the source cannot reach", func(t *testing.T) {
		g := directed("a b c d", es{"a", "b", 1}, es{"c", "d", -1})
		if _, err := g.Dijkstra("a"); !errors.Is(err, ErrNegativeWeight) {
			t.Errorf("Dijkstra(a) error = %v, want ErrNegativeWeight - the contract rejects any negative edge", err)
		}
	})
	t.Run("leaves the graph alone", func(t *testing.T) {
		g := directed("a b c", es{"a", "b", 1}, es{"b", "c", 2})
		g.Dijkstra("a")
		assertGraph(t, g, directed("a b c", es{"a", "b", 1}, es{"b", "c", 2}))
	})
}

func TestShortestPath(t *testing.T) {
	for _, tc := range shortestCases {
		want := distances(tc.g, tc.source)
		for _, to := range tc.g.vertices {
			t.Run(tc.name+" to "+to, func(t *testing.T) {
				path, weight, err := tc.g.ShortestPath(tc.source, to)
				d, reachable := want[to]
				if !reachable {
					if !errors.Is(err, ErrNoPath) {
						t.Errorf("ShortestPath(%s, %s) error = %v, want ErrNoPath", tc.source, to, err)
					}
					return
				}
				if err != nil {
					t.Fatalf("ShortestPath(%s, %s) error = %v, want nil", tc.source, to, err)
				}
				if weight != d {
					t.Errorf("ShortestPath(%s, %s) weight = %d, want %d", tc.source, to, weight, d)
				}
				if len(path) == 0 || path[0] != tc.source || path[len(path)-1] != to {
					t.Fatalf("ShortestPath(%s, %s) path = %v, want it to run from %s to %s", tc.source, to, path, tc.source, to)
				}
				sum := 0
				for i := 1; i < len(path); i++ {
					w, ok := weightOf(tc.g, path[i-1], path[i])
					if !ok {
						t.Fatalf("ShortestPath(%s, %s) path = %v uses %s->%s, which is not an edge", tc.source, to, path, path[i-1], path[i])
					}
					sum += w
				}
				if sum != d {
					t.Errorf("ShortestPath(%s, %s) path = %v weighs %d, want %d", tc.source, to, path, sum, d)
				}
			})
		}
	}

	t.Run("to itself", func(t *testing.T) {
		g := directed("a b", es{"a", "b", 1})
		path, weight, err := g.ShortestPath("a", "a")
		if err != nil || weight != 0 || !slices.Equal(path, []string{"a"}) {
			t.Errorf("ShortestPath(a, a) = %v, %d, %v, want [a], 0, nil", path, weight, err)
		}
	})
	t.Run("directed has no way back", func(t *testing.T) {
		g := directed("a b", es{"a", "b", 1})
		if _, _, err := g.ShortestPath("b", "a"); !errors.Is(err, ErrNoPath) {
			t.Errorf("ShortestPath(b, a) error = %v, want ErrNoPath", err)
		}
	})
	t.Run("absent from", func(t *testing.T) {
		if _, _, err := directed("a").ShortestPath("x", "a"); !errors.Is(err, ErrVertexNotFound) {
			t.Errorf("ShortestPath(x, a) error = %v, want ErrVertexNotFound", err)
		}
	})
	t.Run("absent to", func(t *testing.T) {
		if _, _, err := directed("a").ShortestPath("a", "x"); !errors.Is(err, ErrVertexNotFound) {
			t.Errorf("ShortestPath(a, x) error = %v, want ErrVertexNotFound", err)
		}
	})
	t.Run("negative edge", func(t *testing.T) {
		g := directed("a b", es{"a", "b", -1})
		if _, _, err := g.ShortestPath("a", "b"); !errors.Is(err, ErrNegativeWeight) {
			t.Errorf("ShortestPath(a, b) error = %v, want ErrNegativeWeight", err)
		}
	})
}

func TestString(t *testing.T) {
	for _, tc := range []struct {
		name string
		g    *Graph[string]
		want string
	}{
		{"isolated vertex", directed("a"), "a:"},
		{"directed", directed("a b c", es{"a", "b", 1}, es{"a", "c", 4}), "a: b(1) c(4)\nb:\nc:"},
		{"undirected lists both ends", undirected("a b", es{"a", "b", 2}), "a: b(2)\nb: a(2)"},
		{"order added, not sorted", directed("b a", es{"b", "a", 0}), "b: a(0)\na:"},
		{"self-loop and negative weight", directed("a", es{"a", "a", -3}), "a: a(-3)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.g.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestPrintGraph delegates to String by design - printing the rendered graph is
// the behavior under test - so it goes red if String is broken.
func TestPrintGraph(t *testing.T) {
	for _, tc := range []struct {
		name string
		g    *Graph[string]
		want string
	}{
		{"empty", directed(""), "\n"},
		{"two vertices", directed("a b", es{"a", "b", 1}), "a: b(1)\nb:\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := captureStdout(t, tc.g.PrintGraph); got != tc.want {
				t.Errorf("PrintGraph() wrote %q, want %q", got, tc.want)
			}
		})
	}
}

func ExampleGraph() {
	g := NewDirected[string]()
	g.AddEdge("home", "shop", 4)
	g.AddEdge("home", "park", 1)
	g.AddEdge("park", "shop", 2)

	order, _ := g.BFS("home")
	fmt.Println(order)

	path, weight, _ := g.ShortestPath("home", "shop")
	fmt.Println(path, weight)
	// Output:
	// [home shop park]
	// [home park shop] 3
}
