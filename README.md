# GoLang DSA Playground

A working repository for implementing classic data structures and algorithms from
scratch in **Go** — no library shortcuts, no copied solutions.

The Go sibling of [Python-DSA-Playground](https://github.com/edwinargueta/Python-DSA-Playground).
Same structures, same discipline, a language with different sharp edges.

---

## Why this repo exists

I'm a Senior Lead Software Engineer. Day to day, that role is mostly architecture,
technical direction, code review, and mentorship — the leverage comes from judgment
rather than from writing tree traversals by hand. But the fundamentals are exactly
what that judgment is built on, and they atrophy quietly if you never exercise them.

This repo is deliberate practice against that decay:

- **Better technical judgment.** Choosing between a hash map, a balanced tree, and a
  heap is a design decision with real cost implications. Reasoning fluently about
  complexity, memory layout, and access patterns is what separates a defensible
  architecture from one that merely works today.
- **Sharper code review.** Recognizing an accidental O(n²) or a subtle off-by-one in
  someone else's pull request requires having written — and debugged — those patterns
  yourself.
- **Credible mentorship.** Engineers I support hit these problems in interviews and in
  production. Explaining *why* a red-black tree rebalances the way it does is far more
  useful than pointing at documentation.
- **Staying interview-ready.** Senior roles still assess fundamentals. Keeping this warm
  year-round beats cramming under pressure.

Doing it a second time in Go adds something Python cannot: explicit memory layout,
value versus pointer semantics, and a type system that makes the cost of an
abstraction visible. The same linked list is a different exercise here.

---

## Ground rules

The constraints are the point — they're what make this practice rather than review:

1. **Implement from first principles.** No `container/list`, no `container/heap`, no
   `sort.Search` or `slices.BinarySearch` for the structure being built. The standard
   library is the thing being reimplemented, not the tool used to reimplement it.
2. **Generics, not `any`.** Every structure is parameterized — `[T any]`,
   `[T comparable]`, or `[T cmp.Ordered]` as the operations require. No type
   assertions on the happy path.
3. **Document the complexity.** Each operation states its time and space cost on its
   doc comment, and the reasoning behind it.
4. **Handle the edges deliberately.** Empty structures, single elements, duplicates,
   removing the root — and the Go-specific one: a zero value `T` must never be
   confused with an absent value. Return `(T, bool)` or `(T, error)` and mean it.
5. **Test the behavior, not the implementation.** Tests should survive a rewrite of
   the internals.

Errors are values here. Anything that raises `IndexError` in the Python repo returns
a sentinel error in this one, asserted with `errors.Is` — `panic` is reserved for
unwritten stubs.

---

## Engineering Workflow & AI Acceleration

To maximize implementation speed while maintaining strict code quality:
* **Core Architecture & Logic:** Designed and implemented manually from first principles.
* **Test Case Generation:** Standard assertions and edge-case suites (e.g., boundary
  conditions, random tree insertion sequences) were generated using Claude and verified
  against manual invariants.

---

## Repository structure

Each top-level folder is one self-contained package, with its implementation, its
tests, and its own notes. Packages are independent — there are no cross-imports and
no shared framework, so any one can be read in isolation.

```
GoLang-DSA-Playground/
├── go.mod
├── doublylinkedlist/
│   ├── node.go                       # node primitive
│   ├── doubly_linked_list.go         # the type and its operations
│   └── doubly_linked_list_test.go    # the suite
└── ...                               # further structures as they're built
```

---

## Progress

| Structure | Status | Key operations |
|---|---|---|
| Binary Search Tree | ⚪ Planned | insert, contains, delete, traversals, height, validation |
| Singly Linked List | 🟢 Complete | append, prepend, indexed insert/remove, reversal, cycle detection |
| Doubly Linked List | 🟢 Complete | head and tail pointers, O(1) pop at both ends, reverse traversal |
| Stack & Queue | ⚪ Planned | slice- and node-backed, min-stack |
| Hash Table | ⚪ Planned | separate chaining, open addressing, resize |
| Heap / Priority Queue | ⚪ Planned | sift up/down, heapify, k-largest |
| Graph | ⚪ Planned | BFS, DFS, topological sort, Dijkstra |
| Sorting | ⚪ Planned | merge, quick, heap, counting |
| Dynamic Programming | ⚪ Planned | memoization vs. tabulation, classic problems |

🟢 Complete · 🟡 In progress · ⚪ Planned

---

## Prerequisites

**Go 1.21 or newer** is the only hard requirement — `cmp.Ordered` and the `slices`
package both landed in 1.21. There are no third-party dependencies, so there is
nothing to fetch and no `go.sum` to reconcile.

### Installing the toolchain

| Platform | How |
|---|---|
| macOS (Homebrew) | `brew install go` |
| Linux (Debian/Ubuntu) | `sudo apt install golang-go` — check the version, distro packages lag |
| Any platform | Download the installer from [go.dev/dl](https://go.dev/dl/) |

The official package is the shortest path if you don't already run Homebrew. On
macOS pick the **`darwin-arm64`** build for Apple Silicon (`darwin-amd64` for
Intel); it installs to `/usr/local/go` and puts `go` on the `PATH` of every new
terminal.

Installing Homebrew first is worth it only if you want the rest of its ecosystem:

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

It asks for your login password, and on Apple Silicon it installs to `/opt/homebrew`
and prints a `brew shellenv` line to append to `~/.zprofile`. Skipping that line is
the usual reason `brew: command not found` survives an otherwise successful install.

### Verifying

```bash
go version              # go1.21 or newer
go env GOROOT GOPATH    # GOPATH/bin is where `go install` drops binaries
```

Everything the ground rules ask for — `go build`, `go test`, `go vet`, `gofmt` —
ships with the toolchain. There is no separate test runner, linter, or formatter to
install.

### Optional tooling

| Tool | Install | Why |
|---|---|---|
| `gopls` | editor prompt | Language server: completion, jump-to-definition, inline vet errors |
| `dlv` | editor prompt | Debugger — stepping through a rotation beats `fmt.Println` |
| `gh` | `brew install gh` | GitHub CLI, for cloning and pull requests without the browser |

In VS Code, install the **Go extension** (`golang.Go`) and accept its offer to set up
`gopls` and `dlv` — both land in `$GOPATH/bin`. Turn on format-on-save while you are
there: `gofmt -l .` printing nothing is a standing requirement in this repo, and an
editor enforces it more reliably than memory does.

---

## Running the code

```bash
git clone git@github.com:edwinargueta/GoLang-DSA-Playground.git
cd GoLang-DSA-Playground

go test ./... -v                    # every structure
go test ./doublylinkedlist/ -v      # one structure
go test ./... -run TestPop -v       # one method, everywhere
go vet ./...                        # before every commit
```

Each package includes an `Example` function demonstrating its structure in use.
`go test` runs those too, so the documentation cannot drift from the code.

---

## Big-O complexity reference

Quick reference for the structures and algorithms covered in this repo.
`n` = number of elements, `V` = vertices, `E` = edges, `k` = value range,
`d` = number of digits, `m` = key length.

### Data structures

Average case, with worst case noted where it differs materially:

| Structure | Access | Search | Insert | Delete | Space |
|---|---|---|---|---|---|
| Array (`[N]T`) | O(1) | O(n) | O(n) | O(n) | O(n) |
| Slice (`[]T`) | O(1) | O(n) | O(1)* / O(n) | O(n) | O(n) |
| Singly linked list | O(n) | O(n) | O(1)† | O(1)† | O(n) |
| Doubly linked list | O(n) | O(n) | O(1)† | O(1)† | O(n) |
| Stack | O(n) | O(n) | O(1) | O(1) | O(n) |
| Queue | O(n) | O(n) | O(1) | O(1) | O(n) |
| Map (`map[K]V`) | — | O(1) | O(1) | O(1) | O(n) |
| Binary search tree | O(log n) | O(log n) | O(log n) | O(log n) | O(n) |
| Balanced BST (AVL, red-black) | O(log n) | O(log n) | O(log n) | O(log n) | O(n) |
| Binary heap | O(1)‡ | O(n) | O(log n) | O(log n) | O(n) |
| Trie | — | O(m) | O(m) | O(m) | O(n·m) |

\* Amortized for `append`; O(n) for arbitrary-position insert or a reallocation.
† At the head, or at a node you already hold a pointer to; O(n) if you must find it first.
‡ Peek at min/max only — arbitrary access is O(n).

**Worst cases that matter:**

| Structure | Degrades to | Trigger |
|---|---|---|
| Map | O(n) per operation | Every key colliding into one bucket |
| Binary search tree | O(n) per operation | Sorted or near-sorted insertion order |
| Balanced BST | O(log n) — no degradation | Rebalancing is the guarantee |

### Sorting algorithms

| Algorithm | Best | Average | Worst | Space | Stable |
|---|---|---|---|---|---|
| Bubble sort | O(n) | O(n²) | O(n²) | O(1) | Yes |
| Insertion sort | O(n) | O(n²) | O(n²) | O(1) | Yes |
| Selection sort | O(n²) | O(n²) | O(n²) | O(1) | No |
| Merge sort | O(n log n) | O(n log n) | O(n log n) | O(n) | Yes |
| Quicksort | O(n log n) | O(n log n) | O(n²) | O(log n) | No |
| Heapsort | O(n log n) | O(n log n) | O(n log n) | O(1) | No |
| Counting sort | O(n + k) | O(n + k) | O(n + k) | O(k) | Yes |
| Radix sort | O(d(n + k)) | O(d(n + k)) | O(d(n + k)) | O(n + k) | Yes |
| `slices.Sort` (pdqsort) | O(n) | O(n log n) | O(n log n) | O(log n) | No |

*Stable* means equal elements keep their original relative order — which matters
whenever you sort by one key after already sorting by another. Go makes you choose
explicitly: `slices.Sort` is not stable, `slices.SortStableFunc` is.

### Graph algorithms

| Algorithm | Time | Space | Notes |
|---|---|---|---|
| BFS | O(V + E) | O(V) | Shortest path on unweighted graphs |
| DFS | O(V + E) | O(V) | Cycle detection, connected components |
| Topological sort | O(V + E) | O(V) | DAGs only |
| Dijkstra (binary heap) | O((V + E) log V) | O(V) | No negative edge weights |
| Bellman-Ford | O(V·E) | O(V) | Handles negative edges; detects negative cycles |
| Floyd-Warshall | O(V³) | O(V²) | All-pairs shortest paths |
| Union-Find | O(α(n)) amortized | O(V) | With path compression and union by rank |

α(n) is the inverse Ackermann function — under 5 for any input size that fits in
memory, so effectively constant.

### Graph representations

| Representation | Space | Edge lookup | Iterate neighbors |
|---|---|---|---|
| Adjacency list | O(V + E) | O(degree) | O(degree) |
| Adjacency matrix | O(V²) | O(1) | O(V) |

Adjacency lists win for sparse graphs, which is most real-world graphs. Matrices are
worth it only when the graph is dense or you need constant-time edge queries.

### Go-specific costs

Worth internalizing, since these are where idiomatic-looking Go quietly goes
quadratic — or quietly allocates:

| Operation | Complexity | Note |
|---|---|---|
| `slices.Contains(s, x)` | O(n) | Linear scan |
| `_, ok := m[k]` | O(1) average | The usual fix for the above |
| `append(s, x)` | O(1) amortized | Occasional O(n) reallocate-and-copy |
| `append([]T{x}, s...)` | O(n) | Prepending copies the whole slice |
| `s = append(s[:i], s[i+1:]...)` | O(n) | Deleting shifts every later element |
| `s += "x"` in a loop | O(n²) | Strings are immutable — use `strings.Builder` |
| `for i, v := range s` | O(n) | `v` is a **copy** each iteration |
| `slices.Sort(s)` | O(n log n) | pdqsort; not stable |
| `copy(dst, src)` | O(n) | Bounded by the shorter slice |

Three Go traps with no Python equivalent:

- **`range` copies.** `for _, v := range items` copies each element. For a large
  struct that is a real cost, and writing to `v` mutates nothing. Index or use a
  pointer element type when it matters.
- **Slices share backing arrays.** `b := a[1:3]` aliases `a`. Writing through `b`
  changes `a`, and `append` may or may not, depending on capacity. `slices.Clone`
  when you mean a copy.
- **A `nil` map reads fine and panics on write.** Reading a missing key returns the
  zero value; assigning to a nil map panics. A `nil` slice, by contrast, appends
  happily — the asymmetry catches everyone once.

The classic bug is a `slices.Contains` check inside a loop over the same slice — it
reads as O(n) and runs as O(n²). Swapping the slice for a `map[T]struct{}` is usually
the whole fix.

---

## Choosing the right structure

Complexity tables answer *how fast*; this answers *which one*:

| Requirement | Structure | Why |
|---|---|---|
| Key/value lookup by exact key | Map | O(1) average |
| Sorted order maintained | Balanced BST | O(log n) ops, in-order traversal is sorted |
| Always need the min or max | Heap | O(1) peek, O(log n) extract |
| LIFO | Stack | Natural fit |
| FIFO | Queue | Natural fit |
| Prefix / autocomplete search | Trie | O(m) in key length, not collection size |
| Frequent insert/delete at ends | Doubly linked list | O(1) at both ends |
| Index-based access | Slice | O(1) random access |

The gap between a BST's average and worst case is the entire argument for
self-balancing trees — an unbalanced BST fed sorted input degrades into a linked list,
and O(log n) silently becomes O(n).

Go adds a second axis Python does not have: a slice of values has better cache
locality than a linked list of pointers, and for small `n` that constant factor beats
the asymptotics. Measure before assuming the linked list wins.
