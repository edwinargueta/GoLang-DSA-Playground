// White-box tests for the node-backed queue.
//
//	go test ./queue/ -run TestLinkedQueue -v         # this type
//	go test ./queue/ -run TestLinkedQueueDequeue -v  # one method
//
// Test names are prefixed with the type because two queues share this package and
// Go test function names are package-scoped; captureStdout lives in queue_test.go,
// where one declaration serves both suites.
//
// The oracle walks the chain from head, bounded by maxNodes so a queue wired into a
// cycle fails the suite rather than hanging it. The file is package queue, not
// package queue_test, so those helpers can read head, tail, next and size while the
// fields stay unexported.
//
// assertLinkedQueue spends most of its length on tail, because a stale tail is the
// bug this structure has and values cannot show it. Dequeue the last element while
// leaving tail pointing at the node just removed and the queue reads as empty,
// reports a size of zero and renders correctly - and the next Enqueue links a node
// onto the removed one and loses it. So the helper checks that tail is nil on an
// empty queue, that tail.next is nil, and that tail is the same node the walk from
// head ends on.
//
// Two deliberate exceptions, each noted again above the test itself:
//   - TestLinkedQueueShape guards the shape of the structure rather than a method.
//   - ExampleLinkedQueue exercises NewLinked, Enqueue, PrintQueue, Dequeue and Len
//     together, because a runnable example is by definition an integration.
package queue

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
)

// maxNodes bounds every walking helper, so a chain wired into a cycle fails the suite
// instead of hanging it.
const maxNodes = 1000

// walkFrom collects values by following next from head, independent of any method.
func walkFrom[T comparable](head *node[T], limit int) []T {
	var values []T
	for n := head; n != nil; n = n.next {
		if len(values) > limit {
			panic("walked past the limit - the chain has a cycle")
		}
		values = append(values, n.value)
	}
	return values
}

// buildLinkedQueue returns a queue holding values front to back, wiring the nodes by
// hand rather than calling Enqueue, so every test in this file is independent of the
// implementation and the methods can be written in any order.
func buildLinkedQueue[T comparable](values ...T) *LinkedQueue[T] {
	q := &LinkedQueue[T]{}
	for _, v := range values {
		n := &node[T]{value: v}
		if q.head == nil {
			q.head = n
		} else {
			q.tail.next = n
		}
		q.tail = n
		q.size++
	}
	return q
}

// assertLinkedQueue checks every structural invariant at once: the chain read from
// head, the size counter, both pointers nil when the queue is empty, a nil next on
// the tail, and tail being the node the chain actually ends on. It reads q.size
// directly rather than calling Len, so it never borrows a method to judge another.
// Call it after every mutation, and after every rejected one too - a call that
// returns an error must leave the queue untouched.
func assertLinkedQueue[T comparable](t *testing.T, q *LinkedQueue[T], want []T) {
	t.Helper()

	if got := walkFrom(q.head, maxNodes); !slices.Equal(got, want) {
		t.Errorf("chain from head = %v, want %v", got, want)
	}
	if q.size != len(want) {
		t.Errorf("size field = %d, want %d", q.size, len(want))
	}

	if len(want) == 0 {
		if q.head != nil {
			t.Errorf("head = node(%v), want nil on an empty queue", q.head.value)
		}
		if q.tail != nil {
			t.Errorf("tail = node(%v), want nil on an empty queue - a tail left pointing at a removed node loses the next value enqueued", q.tail.value)
		}
		return
	}

	if q.head == nil || q.tail == nil {
		t.Fatalf("head = %v, tail = %v, want both non-nil for %v", q.head, q.tail, want)
	}
	if q.tail.next != nil {
		t.Errorf("tail.next = node(%v), want nil", q.tail.next.value)
	}

	last := q.head
	for last.next != nil {
		last = last.next
	}
	if q.tail != last {
		t.Errorf("tail is node(%v) but the chain ends on node(%v), want them to be the same node", q.tail.value, last.value)
	}
}

// TestLinkedQueueShape pins the shape of the structure rather than any one method.
// Head and tail and no prev: a queue never removes from the back, which is the one
// operation a forward-only chain cannot do in constant time.
func TestLinkedQueueShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  reflect.Type
		want []string
	}{
		{"queue", reflect.TypeOf(LinkedQueue[int]{}), []string{"head", "tail", "size"}},
		{"node", reflect.TypeOf(node[int]{}), []string{"value", "next"}},
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

// TestLinkedQueueEmpty is the case most implementations get wrong, so every method
// gets a subtest against a freshly built empty queue.
func TestLinkedQueueEmpty(t *testing.T) {
	t.Run("Len", func(t *testing.T) {
		if got := buildLinkedQueue[int]().Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
	})
	t.Run("IsEmpty", func(t *testing.T) {
		if got := buildLinkedQueue[int]().IsEmpty(); !got {
			t.Errorf("IsEmpty() = false, want true")
		}
	})
	t.Run("Enqueue", func(t *testing.T) {
		q := buildLinkedQueue[int]()
		q.Enqueue(10)
		assertLinkedQueue(t, q, []int{10})
	})
	t.Run("Dequeue", func(t *testing.T) {
		q := buildLinkedQueue[int]()
		got, err := q.Dequeue()
		if !errors.Is(err, ErrEmptyQueue) {
			t.Errorf("Dequeue() error = %v, want ErrEmptyQueue", err)
		}
		if got != 0 {
			t.Errorf("Dequeue() = %d, want the zero value", got)
		}
		assertLinkedQueue(t, q, nil)
	})
	t.Run("Peek", func(t *testing.T) {
		q := buildLinkedQueue[int]()
		got, err := q.Peek()
		if !errors.Is(err, ErrEmptyQueue) {
			t.Errorf("Peek() error = %v, want ErrEmptyQueue", err)
		}
		if got != 0 {
			t.Errorf("Peek() = %d, want the zero value", got)
		}
		assertLinkedQueue(t, q, nil)
	})
	t.Run("ToSlice", func(t *testing.T) {
		got := buildLinkedQueue[int]().ToSlice()
		if got == nil {
			t.Fatal("ToSlice() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
		}
	})
	t.Run("String", func(t *testing.T) {
		if got := buildLinkedQueue[int]().String(); got != "nil" {
			t.Errorf("String() = %q, want %q", got, "nil")
		}
	})
}

func TestLinkedQueueNewLinked(t *testing.T) {
	q := NewLinked[int]()
	if q == nil {
		t.Fatal("NewLinked() = nil, want an empty queue")
	}
	assertLinkedQueue(t, q, nil)
}

func TestLinkedQueueLen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
	}{
		{"empty", nil},
		{"one", []int{10}},
		{"two", []int{10, 20}},
		{"many", []int{10, 20, 30, 40}},
		{"duplicates", []int{10, 10, 10}},
		{"zero values", []int{0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildLinkedQueue(tc.start...).Len(); got != len(tc.start) {
				t.Errorf("Len() = %d, want %d", got, len(tc.start))
			}
		})
	}
}

func TestLinkedQueueIsEmpty(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  bool
	}{
		{"empty", nil, true},
		{"one", []int{10}, false},
		{"many", []int{10, 20, 30}, false},
		{"one zero value", []int{0}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildLinkedQueue(tc.start...).IsEmpty(); got != tc.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLinkedQueueEnqueue(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		value int
		want  []int
	}{
		{"onto empty", nil, 10, []int{10}},
		{"onto one", []int{10}, 20, []int{10, 20}},
		{"onto two", []int{10, 20}, 30, []int{10, 20, 30}},
		{"duplicate value", []int{10, 20}, 10, []int{10, 20, 10}},
		{"zero value", []int{10}, 0, []int{10, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := buildLinkedQueue(tc.start...)
			q.Enqueue(tc.value)
			assertLinkedQueue(t, q, tc.want)
		})
	}

	// Onto empty is the case that has to set both pointers; onto one is the case
	// that has to move tail without touching head.
	t.Run("repeated onto empty", func(t *testing.T) {
		q := buildLinkedQueue[int]()
		want := []int{10, 20, 30}
		for i, v := range want {
			q.Enqueue(v)
			assertLinkedQueue(t, q, want[:i+1])
		}
	})
}

func TestLinkedQueueDequeue(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
		rest  []int
	}{
		{"one", []int{10}, 10, nil},
		{"two", []int{10, 20}, 10, []int{20}},
		{"many", []int{10, 20, 30}, 10, []int{20, 30}},
		{"duplicates", []int{10, 10}, 10, []int{10}},
		{"zero value at the front", []int{0, 10}, 0, []int{10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := buildLinkedQueue(tc.start...)
			got, err := q.Dequeue()
			if err != nil {
				t.Fatalf("Dequeue() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Dequeue() = %v, want %v", got, tc.want)
			}
			assertLinkedQueue(t, q, tc.rest)
		})
	}

	// Taking the last element out is the case that has to clear tail as well as
	// head; assertLinkedQueue is what notices when it does not.
	t.Run("drains first in first out", func(t *testing.T) {
		start := []int{10, 20, 30}
		q := buildLinkedQueue(start...)
		for i, want := range start {
			got, err := q.Dequeue()
			if err != nil {
				t.Fatalf("Dequeue() #%d error = %v, want nil", i, err)
			}
			if got != want {
				t.Errorf("Dequeue() #%d = %v, want %v", i, got, want)
			}
			assertLinkedQueue(t, q, start[i+1:])
		}
		if _, err := q.Dequeue(); !errors.Is(err, ErrEmptyQueue) {
			t.Errorf("Dequeue() on the drained queue error = %v, want ErrEmptyQueue", err)
		}
	})

	// The consequence of a stale tail, spelled out: the queue looks right until
	// something is enqueued onto the node that is no longer part of it.
	t.Run("drain then refill", func(t *testing.T) {
		q := buildLinkedQueue(10, 20)
		for i := 0; i < 2; i++ {
			if _, err := q.Dequeue(); err != nil {
				t.Fatalf("Dequeue() #%d error = %v, want nil", i, err)
			}
		}
		assertLinkedQueue(t, q, nil)
		q.Enqueue(30)
		assertLinkedQueue(t, q, []int{30})
		q.Enqueue(40)
		assertLinkedQueue(t, q, []int{30, 40})
	})
}

func TestLinkedQueuePeek(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  int
	}{
		{"one", []int{10}, 10},
		{"two", []int{10, 20}, 10},
		{"many", []int{10, 20, 30}, 10},
		{"duplicates", []int{10, 10}, 10},
		{"zero value at the front", []int{0, 10}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := buildLinkedQueue(tc.start...)
			got, err := q.Peek()
			if err != nil {
				t.Fatalf("Peek() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Peek() = %v, want %v", got, tc.want)
			}
			assertLinkedQueue(t, q, tc.start)
		})
	}

	// Reading the back instead of the front agrees with every one-element case.
	t.Run("reads the front, not the back", func(t *testing.T) {
		q := buildLinkedQueue(10, 20, 30)
		got, err := q.Peek()
		if err != nil {
			t.Fatalf("Peek() error = %v, want nil", err)
		}
		if got != 10 {
			t.Errorf("Peek() = %v, want 10", got)
		}
	})
}

func TestLinkedQueueToSlice(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
	}{
		{"one", []int{10}},
		{"two", []int{10, 20}},
		{"many", []int{10, 20, 30, 40}},
		{"duplicates", []int{10, 10, 10}},
		{"zero values", []int{0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := buildLinkedQueue(tc.start...)
			if got := q.ToSlice(); !slices.Equal(got, tc.start) {
				t.Errorf("ToSlice() = %v, want %v front to back", got, tc.start)
			}
			assertLinkedQueue(t, q, tc.start)
		})
	}

	t.Run("empty is non-nil", func(t *testing.T) {
		got := buildLinkedQueue[int]().ToSlice()
		if got == nil {
			t.Fatal("ToSlice() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
		}
	})
}

func TestLinkedQueueString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "nil"},
		{"one", []int{10}, "10 -> nil"},
		{"two", []int{10, 20}, "10 -> 20 -> nil"},
		{"many", []int{10, 20, 30}, "10 -> 20 -> 30 -> nil"},
		{"zero value", []int{0}, "0 -> nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := buildLinkedQueue(tc.start...)
			if got := q.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
			assertLinkedQueue(t, q, tc.start)
		})
	}
}

// TestLinkedQueuePrintQueue delegates to String by design - printing the rendered
// queue is the behavior under test - so it goes red if String is broken.
func TestLinkedQueuePrintQueue(t *testing.T) {
	for _, tc := range []struct {
		name  string
		start []int
		want  string
	}{
		{"empty", nil, "nil\n"},
		{"one", []int{10}, "10 -> nil\n"},
		{"many", []int{10, 20, 30}, "10 -> 20 -> 30 -> nil\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := buildLinkedQueue(tc.start...)
			got := captureStdout(t, q.PrintQueue)
			if got != tc.want {
				t.Errorf("PrintQueue() wrote %q, want %q", got, tc.want)
			}
			assertLinkedQueue(t, q, tc.start)
		})
	}
}

func ExampleLinkedQueue() {
	q := NewLinked[int]()
	q.Enqueue(10)
	q.Enqueue(20)
	q.Enqueue(30)
	q.PrintQueue()

	front, _ := q.Dequeue()
	fmt.Println(front, q.Len())
	// Output:
	// 10 -> 20 -> 30 -> nil
	// 10 2
}
