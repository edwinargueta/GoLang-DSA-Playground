# Working in this repo

Notes for Claude. This repo is deliberate practice: the owner implements every
structure by hand. Read [README.md](README.md) for the ground rules — the two
that constrain you most are *implement from first principles* (no
`container/list`, no `container/heap`, no `sort.Search` or `slices.BinarySearch`
for the structure being built) and *test the behavior, not the implementation*.

This is the Go sibling of the Python DSA Playground. Same discipline, Go idioms.

## The one rule that matters most

**Never implement a method the owner has not asked you to implement.** Stubs
`panic("not implemented")` on purpose. Writing the solution destroys the point
of the exercise. Write stubs, write tests, explain the failure — but leave the
body alone unless explicitly asked.

## Folder layout

Each top-level folder is one self-contained package. No cross-imports between
structures, no shared helper package — if two structures both need a node, each
declares its own:

```
doublylinkedlist/
├── node.go                      # the node primitive, if one is needed
├── doubly_linked_list.go        # the type, snake_case filename
└── doubly_linked_list_test.go   # the suite
```

Package name is the folder name, lowercase and unpunctuated
(`package doublylinkedlist`), which is why folders carry no separator either.

Unlike the Python repo, tests do **not** have to run from inside their folder —
the module path resolves from anywhere:

```bash
go test ./doublylinkedlist/ -v      # one structure
go test ./... -v                    # everything
go vet ./...                        # before every handoff
gofmt -l .                          # must print nothing
```

## Stub conventions

**One line of doc comment per method. No more.** A stub is a signature and a
contract, not a lesson — long comments crowd out the code, and hints about which
case is tricky are exactly the thinking the owner is here to do.

Go doc comments start with the identifier being declared. Keep the summary, the
return or error contract, and the cost. Drop everything else: worked examples,
algorithm sketches, "remember to..." reminders, and any explanation of *why* a
case is hard.

```go
// Pop removes and returns the tail; ErrEmptyList if empty. O(1) time, O(1) space.
func (l *DoublyLinkedList[T]) Pop() (T, error) {
	panic("not implemented")
}
```

Not this:

```go
// Pop removes and returns the value at the tail.
//
// Returns ErrEmptyList if the list is empty.
//
// This is the operation a singly linked list is bad at: holding tail does
// not help, because you need the node before it and there are no backward
// pointers. A doubly linked list gets it in constant time.
//
// Time:  O(1)
// Space: O(1)
func (l *DoublyLinkedList[T]) Pop() (T, error) {
	panic("not implemented")
}
```

The **type** doc comment is the exception, and the place for anything
structural: the ordering invariant, what happens to duplicates, whether indices
may be negative. Facts that hold across every method belong there once, not
restated on each stub.

`panic("not implemented")` is the whole body. Go treats a panic as terminating,
so no `return` is needed after it and the file still compiles — which means
`go build ./...` stays green while the suite stays red, and a red test always
means the method, never the package.

### Generics

Structures are generic. The element type parameter is `T`:

```go
type DoublyLinkedList[T any] struct{ ... }   // order-free containers
type BST[T cmp.Ordered] struct{ ... }        // needs < and >
```

Use `cmp.Ordered` when the structure compares elements, `comparable` when it
only needs `==`, and `any` when it does neither. If a structure must hold
something unordered in a sorted position, that is a comparator-function design
(`less func(a, b T) bool`) and worth a note in the type doc comment.

### Errors, not exceptions

Go has no exceptions, so the Python API translates like this:

| Python | Go |
|---|---|
| raises `IndexError` | returns `(T, error)` with a sentinel |
| returns `False` for "not found" | returns `(T, bool)`, comma-ok |
| returns `None` | returns nothing |

Each package declares its own sentinels and the tests match with `errors.Is`:

```go
var (
	ErrEmptyList       = errors.New("doublylinkedlist: list is empty")
	ErrIndexOutOfRange = errors.New("doublylinkedlist: index out of range")
)
```

On the error path return the zero value, never a half-built one:

```go
var zero T
return zero, ErrEmptyList
```

**Never `panic` for a caller's bad input** — a panic in a stub means "unwritten",
and overloading it with "you passed a bad index" makes a red test ambiguous.

### Demonstrating the structure

Go has no `__main__` block. The equivalent is a testable `Example` function in
the test file, which `go test` compiles *and* runs, and `go doc` renders:

```go
func ExampleDoublyLinkedList() {
	l := New[int]()
	l.Append(10)
	l.Append(20)
	l.PrintList()
	// Output: 10 <-> 20 <-> nil
}
```

Every implementation file gets one. It replaces the Python `__main__` block and
is strictly better: an out-of-date example fails the build.

---

# How to write a test suite

The goal is that **a green test function means that method is done** — no more,
no less. A suite that fails for the wrong reasons is worse than no suite,
because it destroys the progress signal the owner is working against.

## 1. One test function per method, and it fails only on that method

`func TestPop(t *testing.T)`, `func TestReverse(t *testing.T)`. Cases inside it
are subtests via `t.Run`, so `go test -run TestPop/empty` addresses exactly one.

A test in `TestPop` must fail *only* when `Pop` is wrong. If it also calls
`ToSlice()` to check the result, then `TestPop` goes red because `ToSlice` is
unwritten, and the owner is now debugging the wrong method.

## 2. Give the tests their own way to see inside

The fix is a **test-side oracle**: an unexported helper that inspects the
structure by walking its fields directly, so tests never borrow a method to
verify another method.

This is why the test file is `package doublylinkedlist`, **not**
`package doublylinkedlist_test` — a white-box test file is the deliberate
choice that lets the oracle read `head`, `next`, `prev` and `size` while the
structure keeps them unexported.

```go
// maxNodes bounds every walking helper, so a cycle fails the suite
// instead of hanging it.
const maxNodes = 1000

// walk collects values by following next from head.
func walk[T any](head *node[T], limit int) []T {
	var values []T
	for n := head; n != nil; n = n.next {
		if len(values) > limit {
			panic("walked past the limit - the chain has a cycle")
		}
		values = append(values, n.value)
	}
	return values
}
```

Give any walking helper a `maxNodes` guard if the structure can be wired into a
cycle, or a broken implementation hangs the suite instead of failing it.

## 3. Build fixtures without the implementation

`build()` must construct the structure by wiring nodes directly, not by calling
`Append()` or `Insert()`. Otherwise every test in the file depends on that one
method, and nothing can be implemented out of order.

```go
// build returns a list holding values, wiring both directions by hand.
func build[T any](values ...T) *DoublyLinkedList[T] {
	l := &DoublyLinkedList[T]{}
	for _, v := range values {
		n := &node[T]{value: v}
		if l.head == nil {
			l.head = n
		} else {
			n.prev = l.tail
			l.tail.next = n
		}
		l.tail = n
		l.size++
	}
	return l
}
```

Trade this off per structure, and say which you chose in the comment. Wiring a
tree by hand is more error-prone than the thing it protects against, so a BST
`build()` may legitimately call `Insert()` — but then say so.

## 4. Assert the invariants, not just the values

Most bugs in these structures are a stale pointer or a size counter that
drifted — not a wrong value. One helper, called after every mutation, catches
the whole class of them. `t.Helper()` is mandatory: without it every failure
reports the helper's line instead of the test's.

```go
func assertIntact[T comparable](t *testing.T, l *DoublyLinkedList[T], want []T) {
	t.Helper()
	if got := walk(l.head, maxNodes); !slices.Equal(got, want) {
		t.Errorf("forward chain = %v, want %v", got, want)
	}
	if got := l.Len(); got != len(want) {
		t.Errorf("size = %d, want %d", got, len(want))
	}
	...
}
```

Use Go's `got`/`want` phrasing, and name the thing being reported. `if got != want`
with a bare `t.Error("failed")` makes the owner read the test to find out what
it meant.

## 5. Document the exceptions instead of hiding them

Some coupling is real: a `String()` method delegating to `ToSlice()` *is* the
behavior under test. Keep those, and say so in a comment above the test function
and in the file's doc comment, so a red test is never a surprise:

```go
// TestString delegates to ToSlice by design, so it stays red until
// ToSlice is written.
func TestString(t *testing.T) {
```

## 6. Cover the cases the owner is practicing

Ground rule 4 is *handle the edges deliberately*, so the suite has to exercise
them. Always include a `TestEmpty` function with one subtest per method — the
empty case is the one most implementations get wrong. Then per method:

- **Boundaries** — first and last index, off-by-one on each side, `Len` exactly.
- **Sizes** — empty, one element, two elements, many. One and two elements are
  where head/tail aliasing breaks.
- **The structural cases** — for delete: leaf, one child, two children, and each
  again at the root. For a list: head, middle, tail, only node.
- **Failure paths** — assert the sentinel with `errors.Is` *and* that the
  structure was left unchanged.
- **Zero values** — a `T` that is `0` or `""` must be distinguishable from
  "absent". This is the Go-specific trap the Python suite never had: check the
  `bool`/`error`, never the returned value alone.
- **Duplicates** — whether the structure stores them (linked list) or rejects
  them (BST), test the choice.
- **Round trips** — reverse twice, delete then reinsert, drain then refill.

Table-driven subtests are the idiom where the cases are homogeneous:

```go
for _, tc := range []struct {
	name  string
	index int
	want  int
}{
	{"head", 0, 10},
	{"middle", 2, 30},
	{"tail", 4, 50},
} {
	t.Run(tc.name, func(t *testing.T) { ... })
}
```

Do not force it. A table with one field used by half the rows is worse than
four plain `t.Run` blocks.

## 7. Verify the suite before handing it over

A test suite is code, and an unverified one is worse than useless here. Before
saying it is ready:

1. Write a throwaway reference implementation in the scratchpad, copy the suite
   next to it, and confirm **every test passes**. A test that cannot pass will
   read as the owner's bug.
2. Run each test function against the stubs and confirm it is blocked only by
   its own method — `go test ./pkg/ -run TestPop -v` — this catches accidental
   coupling.
3. Mutate the reference — break one pointer update, drop one `size--` — and
   confirm the suite catches it. A suite that only ever saw correct code has
   proven nothing.
4. `go vet ./...` and `gofmt -l .` clean.
5. Delete the reference implementation. Never commit it, never leave it where it
   could be pasted into the real file.

## Checklist

- [ ] One `Test<Method>` per method; each fails only on its own method
- [ ] Test-side oracle helpers; no method used to verify another
- [ ] Test file is `package <name>` (white-box), so the oracle can read fields
- [ ] `build()` does not depend on the implementation (or a comment says why)
- [ ] Invariant helper called after every mutation, with `t.Helper()`
- [ ] `got`/`want` failure messages that name what broke
- [ ] Documented delegation exceptions
- [ ] Empty / one / two / many, boundaries, failure paths, zero values, duplicates
- [ ] Sentinel errors asserted with `errors.Is`, not string comparison
- [ ] Stub doc comments are one line each, cost included, type doc comment
      carries the invariants
- [ ] An `Example` function per implementation file, with an `// Output:` line
- [ ] File doc comment: how to run it, and what is deliberately coupled
- [ ] Verified green against a scratchpad reference, mutation-checked, deleted
