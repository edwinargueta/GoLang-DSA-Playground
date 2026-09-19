// White-box tests for the binary search tree.
//
//	go test ./binarysearchtree/ -v                 # the whole suite
//	go test ./binarysearchtree/ -run TestDelete -v # one method
//
// Most tests are written against one canonical tree, built by inserting
// 50 30 70 20 40 60 80 10 in that order:
//
//	          50
//	        /    \
//	      30      70
//	     /  \    /  \
//	   20    40 60    80
//	  /
//	10
//
// It is deliberately lopsided: 20 has a single child, 30 and 70 and 50 have two, the
// rest are leaves, and the longest path is three edges. Every structural case Delete
// has to handle appears in it somewhere.
//
// The oracle rests on one theorem: a binary tree is a valid BST over distinct values
// exactly when its in-order walk is strictly increasing. assertIntact walks in-order
// from the root, so it proves the ordering invariant without asking the structure
// anything. In-order alone does not pin the shape, though - every BST over the same
// values produces the same in-order sequence - so assertShape walks pre-order, which
// does pin it, and the tests where the shape is the point call both.
//
// IsValid uses a bounds recursion and the oracle uses the in-order walk on purpose:
// two implementations of the same claim, so a bug would have to occur twice to hide.
//
// The file is package binarysearchtree, not ..._test, so the oracle helpers below can
// read root, value, left, right and size while those fields stay unexported. Every
// assertion goes through those helpers: no method is ever used to verify another, so a
// red test names exactly one broken method.
//
// Three deliberate exceptions, each noted again above the test itself:
//   - TestBinarySearchTree guards the shape of the structure rather than a method.
//   - TestPrintTree delegates to String, because printing the rendered tree is the
//     behavior under test.
//   - ExampleBST exercises New, Insert, PrintTree and InOrder together, because a
//     runnable example is by definition an integration.
package binarysearchtree

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"testing"
)

// maxNodes bounds every walking helper, so a tree wired into a cycle fails the suite
// instead of hanging it.
const maxNodes = 1000

var (
	// canonical is the insertion order of the tree drawn in the file comment.
	canonical = []int{50, 30, 70, 20, 40, 60, 80, 10}
	// canonicalSorted is that tree read in order, and canonicalShape is it read
	// pre-order, which is the sequence that identifies the tree uniquely.
	canonicalSorted = []int{10, 20, 30, 40, 50, 60, 70, 80}
	canonicalShape  = []int{50, 30, 20, 10, 40, 70, 60, 80}
)

// walkInOrder collects values left-root-right, independent of any method.
func walkInOrder[T cmp.Ordered](root *node[T], limit int) []T {
	var values []T
	var visit func(*node[T], int)
	visit = func(n *node[T], depth int) {
		if n == nil {
			return
		}
		if depth > limit || len(values) > limit {
			panic("walked past the limit - the tree has a cycle")
		}
		visit(n.left, depth+1)
		values = append(values, n.value)
		visit(n.right, depth+1)
	}
	visit(root, 0)
	return values
}

// walkPreOrder collects values root-left-right, independent of any method.
func walkPreOrder[T cmp.Ordered](root *node[T], limit int) []T {
	var values []T
	var visit func(*node[T], int)
	visit = func(n *node[T], depth int) {
		if n == nil {
			return
		}
		if depth > limit || len(values) > limit {
			panic("walked past the limit - the tree has a cycle")
		}
		values = append(values, n.value)
		visit(n.left, depth+1)
		visit(n.right, depth+1)
	}
	visit(root, 0)
	return values
}

// build returns a tree holding values, grafting each one with the descent below
// rather than calling Insert, so every test in this file is independent of the
// implementation and the methods can be written in any order. Wiring a tree literally
// is more error-prone than the bug that would protect against, so graft is the
// compromise - and it is a different formulation from Insert, tracking the parent
// node where Insert walks a pointer to the slot it will fill. Duplicates are skipped,
// matching the structure's contract.
func build[T cmp.Ordered](values ...T) *BST[T] {
	tr := &BST[T]{}
	for _, v := range values {
		var grafted bool
		tr.root, grafted = graft(tr.root, v)
		if grafted {
			tr.size++
		}
	}
	return tr
}

// graft hangs a node holding value off root, returning the root to keep and whether
// the value was new.
func graft[T cmp.Ordered](root *node[T], value T) (*node[T], bool) {
	if root == nil {
		return &node[T]{value: value}, true
	}
	for n := root; ; {
		switch {
		case value < n.value:
			if n.left == nil {
				n.left = &node[T]{value: value}
				return root, true
			}
			n = n.left
		case value > n.value:
			if n.right == nil {
				n.right = &node[T]{value: value}
				return root, true
			}
			n = n.right
		default:
			return root, false
		}
	}
}

// branch wires a node literally, ignoring the ordering invariant. TestIsValid needs
// it and rooted below: build cannot produce a tree that breaks the invariant, and a
// broken tree is the only interesting input IsValid has. assertIntact must never be
// called on one.
func branch[T cmp.Ordered](value T, left, right *node[T]) *node[T] {
	return &node[T]{value: value, left: left, right: right}
}

// leaf wires a childless node.
func leaf[T cmp.Ordered](value T) *node[T] {
	return &node[T]{value: value}
}

// rooted wraps an explicitly wired root, counting the nodes rather than trusting a
// caller to keep size honest.
func rooted[T cmp.Ordered](root *node[T]) *BST[T] {
	return &BST[T]{root: root, size: len(walkPreOrder(root, maxNodes))}
}

// assertIntact checks the invariants that hold for every tree at once: the in-order
// sequence, that it is strictly increasing, the size counter, and a nil root when the
// tree is empty. It reads tr.size directly rather than calling Len, so it never
// borrows a method to judge another. Call it after every mutation, and after every
// rejected one too - a call that reports false must leave the tree untouched.
func assertIntact[T cmp.Ordered](t *testing.T, tr *BST[T], want []T) {
	t.Helper()

	got := walkInOrder(tr.root, maxNodes)
	if !slices.Equal(got, want) {
		t.Errorf("in-order walk = %v, want %v", got, want)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			t.Errorf("in-order walk = %v, want strictly increasing - the ordering invariant is broken at %v", got, got[i])
			break
		}
	}
	if tr.size != len(want) {
		t.Errorf("size field = %d, want %d", tr.size, len(want))
	}
	if len(want) == 0 && tr.root != nil {
		t.Errorf("root = node(%v), want nil on an empty tree", tr.root.value)
	}
}

// assertShape pins the tree itself rather than its contents: over distinct values one
// pre-order sequence describes exactly one BST, so this is where "which node got
// promoted" is checked. Values alone cannot prove it - every BST holding 10, 20 and
// 30 walks in order as 10 20 30, whichever of the three is the root.
func assertShape[T cmp.Ordered](t *testing.T, tr *BST[T], want []T) {
	t.Helper()
	if got := walkPreOrder(tr.root, maxNodes); !slices.Equal(got, want) {
		t.Errorf("pre-order walk = %v, want %v", got, want)
	}
}

// captureStdout redirects os.Stdout for the duration of f and returns what was written.
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

// TestBinarySearchTree pins the shape of the structure rather than any one method.
// The node has no parent pointer, and the whole "rewrite the subtree and return its
// new root" design in the type comment is only justified while that stays true.
func TestBinarySearchTree(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  reflect.Type
		want []string
	}{
		{"tree", reflect.TypeOf(BST[int]{}), []string{"root", "size"}},
		{"node", reflect.TypeOf(node[int]{}), []string{"value", "left", "right"}},
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

// TestEmpty is the case most implementations get wrong, so every method gets a
// subtest against a freshly built empty tree.
func TestEmpty(t *testing.T) {
	t.Run("Len", func(t *testing.T) {
		if got := build[int]().Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
	})
	t.Run("Insert", func(t *testing.T) {
		tr := build[int]()
		if got := tr.Insert(10); !got {
			t.Errorf("Insert(10) = false, want true")
		}
		assertIntact(t, tr, []int{10})
		assertShape(t, tr, []int{10})
	})
	t.Run("Contains", func(t *testing.T) {
		if got := build[int]().Contains(10); got {
			t.Errorf("Contains(10) = true, want false")
		}
	})
	t.Run("Delete", func(t *testing.T) {
		tr := build[int]()
		if got := tr.Delete(10); got {
			t.Errorf("Delete(10) = true, want false")
		}
		assertIntact(t, tr, nil)
	})
	t.Run("Min", func(t *testing.T) {
		tr := build[int]()
		got, err := tr.Min()
		if !errors.Is(err, ErrEmptyTree) {
			t.Errorf("Min() error = %v, want ErrEmptyTree", err)
		}
		if got != 0 {
			t.Errorf("Min() = %d, want the zero value", got)
		}
		assertIntact(t, tr, nil)
	})
	t.Run("Max", func(t *testing.T) {
		tr := build[int]()
		got, err := tr.Max()
		if !errors.Is(err, ErrEmptyTree) {
			t.Errorf("Max() error = %v, want ErrEmptyTree", err)
		}
		if got != 0 {
			t.Errorf("Max() = %d, want the zero value", got)
		}
		assertIntact(t, tr, nil)
	})
	t.Run("Height", func(t *testing.T) {
		if got := build[int]().Height(); got != -1 {
			t.Errorf("Height() = %d, want -1", got)
		}
	})
	for _, tc := range []struct {
		name string
		walk func(*BST[int]) []int
	}{
		{"InOrder", (*BST[int]).InOrder},
		{"PreOrder", (*BST[int]).PreOrder},
		{"PostOrder", (*BST[int]).PostOrder},
		{"LevelOrder", (*BST[int]).LevelOrder},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.walk(build[int]())
			if got == nil {
				t.Fatalf("%s() = nil, want an empty non-nil slice", tc.name)
			}
			if len(got) != 0 {
				t.Errorf("%s() = %v, want empty", tc.name, got)
			}
		})
	}
	t.Run("IsValid", func(t *testing.T) {
		if got := build[int]().IsValid(); !got {
			t.Errorf("IsValid() = false, want true - an empty tree is vacuously ordered")
		}
	})
	t.Run("String", func(t *testing.T) {
		if got := build[int]().String(); got != "()" {
			t.Errorf("String() = %q, want %q", got, "()")
		}
	})
}

func TestNew(t *testing.T) {
	tr := New[int]()
	if tr == nil {
		t.Fatal("New() = nil, want an empty tree")
	}
	assertIntact(t, tr, nil)
}

func TestLen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
	}{
		{"empty", nil, 0},
		{"one", []int{10}, 1},
		{"two", []int{10, 20}, 2},
		{"many", canonical, 8},
		{"duplicates are not counted", []int{10, 10, 10}, 1},
		{"zero value counts", []int{0}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := build(tc.start...).Len(); got != tc.want {
				t.Errorf("Len() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestInsert(t *testing.T) {
	for _, tc := range []struct {
		name   string
		start  []int
		value  int
		want   bool
		sorted []int
		shape  []int
	}{
		{"into empty", nil, 10, true, []int{10}, []int{10}},
		{"smaller than the root", []int{10}, 5, true, []int{5, 10}, []int{10, 5}},
		{"larger than the root", []int{10}, 15, true, []int{10, 15}, []int{10, 15}},
		{"left then right", []int{50, 30}, 40, true, []int{30, 40, 50}, []int{50, 30, 40}},
		{"right then left", []int{50, 70}, 60, true, []int{50, 60, 70}, []int{50, 70, 60}},
		{"as a deep leaf", canonical, 45, true,
			[]int{10, 20, 30, 40, 45, 50, 60, 70, 80},
			[]int{50, 30, 20, 10, 40, 45, 70, 60, 80}},
		{"as the new minimum", canonical, 5, true,
			[]int{5, 10, 20, 30, 40, 50, 60, 70, 80},
			[]int{50, 30, 20, 10, 5, 40, 70, 60, 80}},
		{"as the new maximum", canonical, 90, true,
			[]int{10, 20, 30, 40, 50, 60, 70, 80, 90},
			[]int{50, 30, 20, 10, 40, 70, 60, 80, 90}},
		{"duplicate root", canonical, 50, false, canonicalSorted, canonicalShape},
		{"duplicate internal node", canonical, 30, false, canonicalSorted, canonicalShape},
		{"duplicate leaf", canonical, 80, false, canonicalSorted, canonicalShape},
		{"zero value", []int{10, 20}, 0, true, []int{0, 10, 20}, []int{10, 0, 20}},
		{"negative value", []int{0}, -5, true, []int{-5, 0}, []int{0, -5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := build(tc.start...)
			if got := tr.Insert(tc.value); got != tc.want {
				t.Errorf("Insert(%d) = %v, want %v", tc.value, got, tc.want)
			}
			assertIntact(t, tr, tc.sorted)
			assertShape(t, tr, tc.shape)
		})
	}

	t.Run("repeated inserts build the canonical tree", func(t *testing.T) {
		tr := build[int]()
		for i, v := range canonical {
			if got := tr.Insert(v); !got {
				t.Fatalf("Insert(%d) = false, want true", v)
			}
			want := slices.Clone(canonical[:i+1])
			slices.Sort(want)
			assertIntact(t, tr, want)
		}
		assertShape(t, tr, canonicalShape)
	})

	// Not a wart to be fixed - the degeneracy is the reason balanced trees exist, and
	// pinning it here means a later AVL version has something concrete to beat.
	t.Run("sorted input degenerates into a right spine", func(t *testing.T) {
		tr := build[int]()
		for _, v := range []int{10, 20, 30, 40} {
			tr.Insert(v)
		}
		assertIntact(t, tr, []int{10, 20, 30, 40})
		assertShape(t, tr, []int{10, 20, 30, 40})
	})

	t.Run("the zero value is a real element", func(t *testing.T) {
		tr := build[int]()
		if got := tr.Insert(0); !got {
			t.Fatalf("Insert(0) into an empty tree = false, want true")
		}
		if got := tr.Insert(0); got {
			t.Errorf("Insert(0) again = true, want false - a stored zero must not read as an empty slot")
		}
		assertIntact(t, tr, []int{0})
	})
}

func TestContains(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		value int
		want  bool
	}{
		{"root", canonical, 50, true},
		{"left child", canonical, 30, true},
		{"right child", canonical, 70, true},
		{"deepest leaf", canonical, 10, true},
		{"rightmost leaf", canonical, 80, true},
		{"only node", []int{10}, 10, true},
		{"absent between two values", canonical, 45, false},
		{"absent below the minimum", canonical, 5, false},
		{"absent above the maximum", canonical, 90, false},
		{"absent from one node", []int{10}, 99, false},
		{"zero value present", []int{10, 0, 20}, 0, true},
		{"zero value absent", []int{10, 20}, 0, false},
		{"negative present", []int{0, -5}, -5, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := build(tc.start...)
			if got := tr.Contains(tc.value); got != tc.want {
				t.Errorf("Contains(%d) = %v, want %v", tc.value, got, tc.want)
			}
			want := slices.Clone(tc.start)
			slices.Sort(want)
			assertIntact(t, tr, slices.Compact(want))
		})
	}
}

func TestDelete(t *testing.T) {
	for _, tc := range []struct {
		name   string
		start  []int
		value  int
		want   bool
		sorted []int
		shape  []int
	}{
		{"leaf on the left", canonical, 10, true,
			[]int{20, 30, 40, 50, 60, 70, 80},
			[]int{50, 30, 20, 40, 70, 60, 80}},
		{"leaf on the right", canonical, 80, true,
			[]int{10, 20, 30, 40, 50, 60, 70},
			[]int{50, 30, 20, 10, 40, 70, 60}},
		{"one child, promoted in place", canonical, 20, true,
			[]int{10, 30, 40, 50, 60, 70, 80},
			[]int{50, 30, 10, 40, 70, 60, 80}},
		{"two children, successor is the right child", canonical, 30, true,
			[]int{10, 20, 40, 50, 60, 70, 80},
			[]int{50, 40, 20, 10, 70, 60, 80}},
		{"two children, deeper in the tree", canonical, 70, true,
			[]int{10, 20, 30, 40, 50, 60, 80},
			[]int{50, 30, 20, 10, 40, 80, 60}},
		{"root with two children", canonical, 50, true,
			[]int{10, 20, 30, 40, 60, 70, 80},
			[]int{60, 30, 20, 10, 40, 70, 80}},
		{"only node", []int{10}, 10, true, nil, nil},
		{"root with only a left child", []int{20, 10}, 20, true, []int{10}, []int{10}},
		{"root with only a right child", []int{10, 20}, 10, true, []int{20}, []int{20}},
		{"successor carries a right child", []int{50, 30, 70, 60, 80, 65}, 50, true,
			[]int{30, 60, 65, 70, 80},
			[]int{60, 30, 70, 65, 80}},
		{"zero value", []int{10, 0, 20}, 0, true, []int{10, 20}, []int{10, 20}},
		{"absent between two values", canonical, 45, false, canonicalSorted, canonicalShape},
		{"absent below the minimum", canonical, 5, false, canonicalSorted, canonicalShape},
		{"absent above the maximum", canonical, 90, false, canonicalSorted, canonicalShape},
		{"absent zero value", []int{10, 20}, 0, false, []int{10, 20}, []int{10, 20}},
		{"absent from empty", nil, 10, false, nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := build(tc.start...)
			if got := tr.Delete(tc.value); got != tc.want {
				t.Errorf("Delete(%d) = %v, want %v", tc.value, got, tc.want)
			}
			assertIntact(t, tr, tc.sorted)
			assertShape(t, tr, tc.shape)
		})
	}

	t.Run("twice reports false the second time", func(t *testing.T) {
		tr := build(canonical...)
		if got := tr.Delete(30); !got {
			t.Fatalf("Delete(30) = false, want true")
		}
		if got := tr.Delete(30); got {
			t.Errorf("Delete(30) again = true, want false")
		}
		assertIntact(t, tr, []int{10, 20, 40, 50, 60, 70, 80})
	})

	// A single deletion can look right and still have dropped or duplicated a node
	// somewhere below. Draining the whole tree is what catches that.
	t.Run("drains to empty", func(t *testing.T) {
		tr := build(canonical...)
		want := slices.Clone(canonicalSorted)
		for _, v := range []int{50, 10, 70, 30, 80, 20, 60, 40} {
			if got := tr.Delete(v); !got {
				t.Fatalf("Delete(%d) = false, want true", v)
			}
			want = slices.Delete(want, slices.Index(want, v), slices.Index(want, v)+1)
			assertIntact(t, tr, want)
		}
		if tr.root != nil {
			t.Errorf("root = node(%v), want nil after draining", tr.root.value)
		}
	})

	t.Run("delete then reinsert restores the sequence", func(t *testing.T) {
		tr := build(canonical...)
		if got := tr.Delete(30); !got {
			t.Fatalf("Delete(30) = false, want true")
		}
		assertIntact(t, tr, []int{10, 20, 40, 50, 60, 70, 80})

		// Grafted by hand rather than by Insert, so this stays a test of Delete.
		tr.root, _ = graft(tr.root, 30)
		tr.size++
		assertIntact(t, tr, canonicalSorted)
	})
}

func TestMin(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
	}{
		{"canonical", canonical, 10},
		{"only node", []int{10}, 10},
		{"left spine", []int{30, 20, 10}, 10},
		{"right spine", []int{10, 20, 30}, 10},
		{"minimum is the root", []int{10, 20, 30, 15}, 10},
		{"zero value is the minimum", []int{0, 10, 20}, 0},
		{"negative minimum", []int{10, -5, 20}, -5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := build(tc.start...)
			got, err := tr.Min()
			if err != nil {
				t.Fatalf("Min() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Min() = %d, want %d", got, tc.want)
			}
			want := slices.Clone(tc.start)
			slices.Sort(want)
			assertIntact(t, tr, want)
		})
	}
}

func TestMax(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
	}{
		{"canonical", canonical, 80},
		{"only node", []int{10}, 10},
		{"left spine", []int{30, 20, 10}, 30},
		{"right spine", []int{10, 20, 30}, 30},
		{"maximum is the root", []int{30, 20, 10, 25}, 30},
		{"zero value is the maximum", []int{0, -10, -20}, 0},
		{"all negative", []int{-10, -5, -20}, -5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := build(tc.start...)
			got, err := tr.Max()
			if err != nil {
				t.Fatalf("Max() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Max() = %d, want %d", got, tc.want)
			}
			want := slices.Clone(tc.start)
			slices.Sort(want)
			assertIntact(t, tr, want)
		})
	}
}

func TestHeight(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
	}{
		{"empty", nil, -1},
		{"one node", []int{10}, 0},
		{"root and a left child", []int{20, 10}, 1},
		{"root and both children", []int{20, 10, 30}, 1},
		{"balanced seven", []int{40, 20, 60, 10, 30, 50, 70}, 2},
		{"canonical", canonical, 3},
		{"right spine of five", []int{10, 20, 30, 40, 50}, 4},
		{"left spine of four", []int{40, 30, 20, 10}, 3},
		{"one long branch", []int{50, 30, 70, 10}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := build(tc.start...)
			if got := tr.Height(); got != tc.want {
				t.Errorf("Height() = %d, want %d", got, tc.want)
			}
			want := slices.Clone(tc.start)
			slices.Sort(want)
			assertIntact(t, tr, want)
		})
	}
}

func TestInOrder(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  []int
	}{
		{"one node", []int{10}, []int{10}},
		{"root and a left child", []int{20, 10}, []int{10, 20}},
		{"root and a right child", []int{10, 20}, []int{10, 20}},
		{"balanced seven", []int{40, 20, 60, 10, 30, 50, 70}, []int{10, 20, 30, 40, 50, 60, 70}},
		{"canonical", canonical, canonicalSorted},
		{"right spine", []int{10, 20, 30}, []int{10, 20, 30}},
		{"zero and negative", []int{0, -10, 10}, []int{-10, 0, 10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := build(tc.start...)
			if got := tr.InOrder(); !slices.Equal(got, tc.want) {
				t.Errorf("InOrder() = %v, want %v", got, tc.want)
			}
			assertIntact(t, tr, tc.want)
		})
	}

	t.Run("result is a copy", func(t *testing.T) {
		tr := build(canonical...)
		got := tr.InOrder()
		got[0] = 99
		assertIntact(t, tr, canonicalSorted)
	})
}

func TestPreOrder(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  []int
	}{
		{"one node", []int{10}, []int{10}},
		{"root and a left child", []int{20, 10}, []int{20, 10}},
		{"root and a right child", []int{10, 20}, []int{10, 20}},
		{"balanced seven", []int{40, 20, 60, 10, 30, 50, 70}, []int{40, 20, 10, 30, 60, 50, 70}},
		{"canonical", canonical, canonicalShape},
		{"left spine", []int{30, 20, 10}, []int{30, 20, 10}},
		{"lopsided", []int{50, 30, 20, 70}, []int{50, 30, 20, 70}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := build(tc.start...)
			if got := tr.PreOrder(); !slices.Equal(got, tc.want) {
				t.Errorf("PreOrder() = %v, want %v", got, tc.want)
			}
			want := slices.Clone(tc.start)
			slices.Sort(want)
			assertIntact(t, tr, want)
		})
	}
}

func TestPostOrder(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  []int
	}{
		{"one node", []int{10}, []int{10}},
		{"root and a left child", []int{20, 10}, []int{10, 20}},
		{"root and a right child", []int{10, 20}, []int{20, 10}},
		{"balanced seven", []int{40, 20, 60, 10, 30, 50, 70}, []int{10, 30, 20, 50, 70, 60, 40}},
		{"canonical", canonical, []int{10, 20, 40, 30, 60, 80, 70, 50}},
		{"right spine", []int{10, 20, 30}, []int{30, 20, 10}},
		{"lopsided", []int{50, 30, 20, 70}, []int{20, 30, 70, 50}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := build(tc.start...)
			if got := tr.PostOrder(); !slices.Equal(got, tc.want) {
				t.Errorf("PostOrder() = %v, want %v", got, tc.want)
			}
			want := slices.Clone(tc.start)
			slices.Sort(want)
			assertIntact(t, tr, want)
		})
	}
}

func TestLevelOrder(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  []int
	}{
		{"one node", []int{10}, []int{10}},
		{"root and a left child", []int{20, 10}, []int{20, 10}},
		{"root and a right child", []int{10, 20}, []int{10, 20}},
		{"balanced seven", []int{40, 20, 60, 10, 30, 50, 70}, []int{40, 20, 60, 10, 30, 50, 70}},
		{"canonical", canonical, []int{50, 30, 70, 20, 40, 60, 80, 10}},
		{"right spine", []int{10, 20, 30}, []int{10, 20, 30}},
		// Insertion order here is 50 30 20 70, so a level walk that happened to echo
		// it would be wrong.
		{"rows do not follow insertion order", []int{50, 30, 20, 70}, []int{50, 30, 70, 20}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := build(tc.start...)
			if got := tr.LevelOrder(); !slices.Equal(got, tc.want) {
				t.Errorf("LevelOrder() = %v, want %v", got, tc.want)
			}
			want := slices.Clone(tc.start)
			slices.Sort(want)
			assertIntact(t, tr, want)
		})
	}
}

// TestIsValid is the one test that feeds the structure trees the rest of the suite
// cannot produce, so the malformed cases are wired by hand with branch and leaf -
// and assertIntact is deliberately not called on them, since breaking the invariant
// it checks is the whole point.
func TestIsValid(t *testing.T) {
	for _, tc := range []struct {
		name string
		tr   *BST[int]
		want bool
	}{
		{"empty", build[int](), true},
		{"one node", build(10), true},
		{"canonical", build(canonical...), true},
		{"left spine", build(30, 20, 10), true},
		{"right spine", build(10, 20, 30), true},
		{"balanced seven", build(40, 20, 60, 10, 30, 50, 70), true},
		{"hand-wired and correct", rooted(branch(20,
			branch(10, leaf(5), leaf(15)),
			branch(30, leaf(25), leaf(35)))), true},

		{"left child is larger", rooted(branch(10, leaf(20), nil)), false},
		{"right child is smaller", rooted(branch(10, nil, leaf(5))), false},
		// The case a check that only compares parent to child gets wrong: 25 is
		// correctly to the right of 10 and still in the wrong subtree of 20.
		{"grandchild above the root on the left", rooted(branch(20,
			branch(10, nil, leaf(25)),
			leaf(30))), false},
		{"grandchild below the root on the right", rooted(branch(20,
			leaf(10),
			branch(30, leaf(15), nil))), false},
		// The bound is strict, so a grandchild landing exactly on it is out too.
		{"grandchild equal to the root", rooted(branch(20,
			branch(10, nil, leaf(20)),
			nil)), false},
		{"duplicate on the left", rooted(branch(10, leaf(10), nil)), false},
		{"duplicate on the right", rooted(branch(10, nil, leaf(10))), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tr.IsValid(); got != tc.want {
				t.Errorf("IsValid() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "()"},
		{"one node", []int{10}, "10"},
		{"left child only", []int{10, 5}, "10(5, .)"},
		{"right child only", []int{10, 15}, "10(., 15)"},
		{"both children", []int{10, 5, 15}, "10(5, 15)"},
		{"canonical", canonical, "50(30(20(10, .), 40), 70(60, 80))"},
		{"zero value", []int{0}, "0"},
		{"negative value", []int{0, -5}, "0(-5, .)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := build(tc.start...)
			if got := tr.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
			want := slices.Clone(tc.start)
			slices.Sort(want)
			assertIntact(t, tr, want)
		})
	}
}

// TestPrintTree delegates to String by design - printing the rendered tree is the
// behavior under test - so it goes red if String is broken.
func TestPrintTree(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "()\n"},
		{"one node", []int{10}, "10\n"},
		{"canonical", canonical, "50(30(20(10, .), 40), 70(60, 80))\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := build(tc.start...)
			got := captureStdout(t, tr.PrintTree)
			if got != tc.want {
				t.Errorf("PrintTree() wrote %q, want %q", got, tc.want)
			}
			want := slices.Clone(tc.start)
			slices.Sort(want)
			assertIntact(t, tr, want)
		})
	}
}

func ExampleBST() {
	tr := New[int]()
	for _, v := range []int{50, 30, 70, 20, 40} {
		tr.Insert(v)
	}
	tr.PrintTree()
	fmt.Println(tr.InOrder())
	// Output:
	// 50(30(20, 40), 70)
	// [20 30 40 50 70]
}
