// White-box tests for the ring-backed queue.
//
//	go test ./queue/ -v                        # both queues in the package
//	go test ./queue/ -run TestQueueDequeue -v  # one method
//
// Two queues share this package and Go test function names are package-scoped, so
// every test here is prefixed with the type it covers: TestQueueDequeue is this
// file's, TestLinkedQueueDequeue belongs to the other. captureStdout is declared
// here and used by both files, since one declaration per package is all Go allows.
//
// The oracle reads the ring directly. buildQueue takes the capacity and the head
// index as arguments, which is the only way to hand a test a queue whose live window
// already wraps past the end of the array - no sequence of calls could set that up
// while Enqueue and Dequeue are still stubs, and wraparound is where this structure
// actually goes wrong. Every table below therefore states the layout, not just the
// contents.
//
// assertQueue checks one invariant the values alone cannot show: every cell outside
// the live window holds the zero value. That catches a Dequeue that advances head
// without clearing the cell it left - which for a queue of ints is invisible and for
// a queue of pointers is an unbounded leak - and it catches a grow that copies the
// ring wholesale instead of copying the live window.
//
// Two deliberate exceptions, each noted again above the test itself:
//   - TestQueueShape guards the shape of the structure rather than a method.
//   - ExampleQueue exercises New, Enqueue, PrintQueue, Dequeue and Len together,
//     because a runnable example is by definition an integration.
package queue

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"testing"
)

// buildQueue lays values out front to back in a ring of the given capacity, with the
// front at head, writing the cells directly rather than calling Enqueue, so every
// test in this file is independent of the implementation and the methods can be
// written in any order. A capacity of zero builds the zero Queue, whose ring is nil
// until the first Enqueue allocates one.
func buildQueue[T comparable](capacity, head int, values ...T) *Queue[T] {
	if capacity < len(values) {
		panic("buildQueue: capacity below the number of values")
	}
	q := &Queue[T]{size: len(values)}
	if capacity == 0 {
		return q
	}
	q.items = make([]T, capacity)
	q.head = head % capacity
	for i, v := range values {
		q.items[(q.head+i)%capacity] = v
	}
	return q
}

// queueOf is buildQueue for the cases where the layout is not the point: the front
// at zero and two spare cells, so nothing is on the edge of a grow by accident.
func queueOf[T comparable](values ...T) *Queue[T] {
	return buildQueue(len(values)+2, 0, values...)
}

// ringWalk reads the live window front to back, independent of any method.
func ringWalk[T comparable](q *Queue[T]) []T {
	if len(q.items) == 0 {
		return nil
	}
	var values []T
	for i := 0; i < q.size; i++ {
		values = append(values, q.items[(q.head+i)%len(q.items)])
	}
	return values
}

// assertQueue checks every structural invariant at once: the live window read front
// to back, the size counter, that size fits inside the ring, that head indexes the
// ring, and that every cell outside the window holds the zero value. It reads q.size
// directly rather than calling Len, so it never borrows a method to judge another.
// Call it after every mutation, and after every rejected one too - a call that
// returns an error must leave the queue untouched.
func assertQueue[T comparable](t *testing.T, q *Queue[T], want []T) {
	t.Helper()

	if q.size != len(want) {
		t.Errorf("size field = %d, want %d", q.size, len(want))
	}
	if q.size > len(q.items) {
		t.Fatalf("size = %d, want no more than the ring's %d cells", q.size, len(q.items))
	}
	if len(q.items) == 0 {
		if len(want) != 0 {
			t.Errorf("ring is unallocated, want it to hold %v", want)
		}
		if q.head != 0 {
			t.Errorf("head = %d, want 0 while the ring is unallocated", q.head)
		}
		return
	}
	if q.head < 0 || q.head >= len(q.items) {
		t.Fatalf("head = %d, want an index into the ring's %d cells", q.head, len(q.items))
	}
	if got := ringWalk(q); !slices.Equal(got, want) {
		t.Errorf("ring read front to back = %v, want %v", got, want)
	}

	live := make([]bool, len(q.items))
	for i := 0; i < q.size; i++ {
		live[(q.head+i)%len(q.items)] = true
	}
	var zero T
	for i, v := range q.items {
		if !live[i] && v != zero {
			t.Errorf("ring cell %d holds %v outside the live window, want the zero value - a dequeued element is still reachable", i, v)
		}
	}
}

// captureStdout redirects os.Stdout for the duration of f and returns what was
// written. Shared by both queue suites in this package.
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

// TestQueueShape pins the shape of the structure rather than any one method. A count
// and not a tail index: with two indices, head == tail describes an empty ring and a
// full one alike, and the ambiguity has to be bought off with a wasted cell.
func TestQueueShape(t *testing.T) {
	var got []string
	for _, f := range reflect.VisibleFields(reflect.TypeOf(Queue[int]{})) {
		got = append(got, f.Name)
	}
	if want := []string{"items", "head", "size"}; !slices.Equal(got, want) {
		t.Errorf("Queue fields = %v, want %v", got, want)
	}
}

// TestQueueEmpty is the case most implementations get wrong, so every method gets a
// subtest against a freshly built empty queue - the zero Queue, whose ring is still
// nil, since that is the emptiest one there is.
func TestQueueEmpty(t *testing.T) {
	t.Run("Len", func(t *testing.T) {
		if got := buildQueue[int](0, 0).Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
	})
	t.Run("IsEmpty", func(t *testing.T) {
		if got := buildQueue[int](0, 0).IsEmpty(); !got {
			t.Errorf("IsEmpty() = false, want true")
		}
	})
	t.Run("Enqueue allocates the ring", func(t *testing.T) {
		q := buildQueue[int](0, 0)
		q.Enqueue(10)
		assertQueue(t, q, []int{10})
	})
	t.Run("Dequeue", func(t *testing.T) {
		q := buildQueue[int](0, 0)
		got, err := q.Dequeue()
		if !errors.Is(err, ErrEmptyQueue) {
			t.Errorf("Dequeue() error = %v, want ErrEmptyQueue", err)
		}
		if got != 0 {
			t.Errorf("Dequeue() = %d, want the zero value", got)
		}
		assertQueue(t, q, nil)
	})
	t.Run("Peek", func(t *testing.T) {
		q := buildQueue[int](0, 0)
		got, err := q.Peek()
		if !errors.Is(err, ErrEmptyQueue) {
			t.Errorf("Peek() error = %v, want ErrEmptyQueue", err)
		}
		if got != 0 {
			t.Errorf("Peek() = %d, want the zero value", got)
		}
		assertQueue(t, q, nil)
	})
	t.Run("ToSlice", func(t *testing.T) {
		got := buildQueue[int](0, 0).ToSlice()
		if got == nil {
			t.Fatal("ToSlice() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
		}
	})
	t.Run("String", func(t *testing.T) {
		if got := buildQueue[int](0, 0).String(); got != "nil" {
			t.Errorf("String() = %q, want %q", got, "nil")
		}
	})
	// An allocated ring that has been drained is a second kind of empty, and the one
	// where a head left mid-ring can still be wrong.
	t.Run("drained ring", func(t *testing.T) {
		q := buildQueue[int](4, 3)
		if got := q.Len(); got != 0 {
			t.Errorf("Len() = %d, want 0", got)
		}
		if got := q.IsEmpty(); !got {
			t.Errorf("IsEmpty() = false, want true")
		}
		if _, err := q.Dequeue(); !errors.Is(err, ErrEmptyQueue) {
			t.Errorf("Dequeue() error = %v, want ErrEmptyQueue", err)
		}
		assertQueue(t, q, nil)
	})
}

func TestQueueNew(t *testing.T) {
	q := New[int]()
	if q == nil {
		t.Fatal("New() = nil, want an empty queue")
	}
	assertQueue(t, q, nil)
}

func TestQueueLen(t *testing.T) {
	for _, tc := range []struct {
		name  string
		queue *Queue[int]
		want  int
	}{
		{"unallocated", buildQueue[int](0, 0), 0},
		{"allocated and empty", buildQueue[int](4, 2), 0},
		{"one", queueOf(10), 1},
		{"two", queueOf(10, 20), 2},
		{"many", queueOf(10, 20, 30, 40), 4},
		{"wrapped", buildQueue(4, 3, 10, 20, 30), 3},
		{"full ring", buildQueue(3, 0, 10, 20, 30), 3},
		{"duplicates", queueOf(10, 10, 10), 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.queue.Len(); got != tc.want {
				t.Errorf("Len() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestQueueIsEmpty(t *testing.T) {
	for _, tc := range []struct {
		name  string
		queue *Queue[int]
		want  bool
	}{
		{"unallocated", buildQueue[int](0, 0), true},
		{"allocated and empty", buildQueue[int](4, 2), true},
		{"one", queueOf(10), false},
		{"one zero value", queueOf(0), false},
		{"wrapped", buildQueue(4, 3, 10, 20), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.queue.IsEmpty(); got != tc.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestQueueEnqueue(t *testing.T) {
	for _, tc := range []struct {
		name  string
		queue *Queue[int]
		value int
		want  []int
	}{
		{"into the unallocated ring", buildQueue[int](0, 0), 10, []int{10}},
		{"into an allocated empty ring", buildQueue[int](4, 2), 10, []int{10}},
		{"onto one", queueOf(10), 20, []int{10, 20}},
		{"onto many", queueOf(10, 20), 30, []int{10, 20, 30}},
		{"into the last free cell", buildQueue(3, 0, 10, 20), 30, []int{10, 20, 30}},
		{"wrapping past the end", buildQueue(3, 2, 10), 20, []int{10, 20}},
		{"into a wrapped ring", buildQueue(4, 3, 10, 20), 30, []int{10, 20, 30}},
		{"duplicate value", queueOf(10), 10, []int{10, 10}},
		{"zero value", queueOf(10), 0, []int{10, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.queue.Enqueue(tc.value)
			assertQueue(t, tc.queue, tc.want)
		})
	}

	// Growing is where the ring is at its most fragile: the live window may already
	// be split across the end of the array, and copying the array rather than the
	// window puts the elements back in the wrong order.
	t.Run("grows a full ring", func(t *testing.T) {
		q := buildQueue(3, 0, 10, 20, 30)
		q.Enqueue(40)
		assertQueue(t, q, []int{10, 20, 30, 40})
		if len(q.items) < 4 {
			t.Errorf("ring holds %d cells, want at least 4 after the grow", len(q.items))
		}
	})

	t.Run("grows a full wrapped ring", func(t *testing.T) {
		q := buildQueue(3, 2, 10, 20, 30)
		q.Enqueue(40)
		assertQueue(t, q, []int{10, 20, 30, 40})
	})

	t.Run("grows repeatedly from empty", func(t *testing.T) {
		q := buildQueue[int](0, 0)
		want := make([]int, 0, 100)
		for i := 0; i < 100; i++ {
			q.Enqueue(i)
			want = append(want, i)
			assertQueue(t, q, want)
		}
	})
}

func TestQueueDequeue(t *testing.T) {
	for _, tc := range []struct {
		name  string
		queue *Queue[int]
		want  int
		rest  []int
	}{
		{"one", queueOf(10), 10, nil},
		{"two", queueOf(10, 20), 10, []int{20}},
		{"many", queueOf(10, 20, 30), 10, []int{20, 30}},
		{"full ring", buildQueue(3, 0, 10, 20, 30), 10, []int{20, 30}},
		{"wrapped", buildQueue(4, 3, 10, 20, 30), 10, []int{20, 30}},
		{"head wraps past the end", buildQueue(3, 2, 10, 20), 10, []int{20}},
		{"duplicates", queueOf(10, 10), 10, []int{10}},
		{"zero value at the front", queueOf(0, 10), 0, []int{10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.queue.Dequeue()
			if err != nil {
				t.Fatalf("Dequeue() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Dequeue() = %v, want %v", got, tc.want)
			}
			assertQueue(t, tc.queue, tc.rest)
		})
	}

	t.Run("drains first in first out", func(t *testing.T) {
		start := []int{10, 20, 30}
		q := queueOf(start...)
		for i, want := range start {
			got, err := q.Dequeue()
			if err != nil {
				t.Fatalf("Dequeue() #%d error = %v, want nil", i, err)
			}
			if got != want {
				t.Errorf("Dequeue() #%d = %v, want %v", i, got, want)
			}
			assertQueue(t, q, start[i+1:])
		}
		if _, err := q.Dequeue(); !errors.Is(err, ErrEmptyQueue) {
			t.Errorf("Dequeue() on the drained queue error = %v, want ErrEmptyQueue", err)
		}
	})

	// The leak made visible: with pointers, a cell Dequeue failed to zero is a live
	// reference to an object the caller believes it handed back.
	t.Run("releases the dequeued element", func(t *testing.T) {
		first, second := new(int), new(int)
		q := queueOf(first, second)
		got, err := q.Dequeue()
		if err != nil {
			t.Fatalf("Dequeue() error = %v, want nil", err)
		}
		if got != first {
			t.Errorf("Dequeue() returned the wrong pointer")
		}
		assertQueue(t, q, []*int{second})
	})
}

func TestQueuePeek(t *testing.T) {
	for _, tc := range []struct {
		name  string
		queue *Queue[int]
		want  int
		rest  []int
	}{
		{"one", queueOf(10), 10, []int{10}},
		{"many", queueOf(10, 20, 30), 10, []int{10, 20, 30}},
		{"wrapped", buildQueue(4, 3, 10, 20, 30), 10, []int{10, 20, 30}},
		{"full ring", buildQueue(3, 0, 10, 20, 30), 10, []int{10, 20, 30}},
		{"duplicates", queueOf(10, 10), 10, []int{10, 10}},
		{"zero value at the front", queueOf(0, 10), 0, []int{0, 10}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.queue.Peek()
			if err != nil {
				t.Fatalf("Peek() error = %v, want nil", err)
			}
			if got != tc.want {
				t.Errorf("Peek() = %v, want %v", got, tc.want)
			}
			assertQueue(t, tc.queue, tc.rest)
		})
	}

	// Peek must read the front, not the first cell of the array. On an unwrapped
	// ring those are the same cell and the distinction never shows.
	t.Run("reads the front of a wrapped ring", func(t *testing.T) {
		q := buildQueue(4, 2, 10, 20, 30)
		got, err := q.Peek()
		if err != nil {
			t.Fatalf("Peek() error = %v, want nil", err)
		}
		if got != 10 {
			t.Errorf("Peek() = %v, want 10", got)
		}
	})
}

func TestQueueToSlice(t *testing.T) {
	for _, tc := range []struct {
		name  string
		queue *Queue[int]
		want  []int
	}{
		{"one", queueOf(10), []int{10}},
		{"two", queueOf(10, 20), []int{10, 20}},
		{"many", queueOf(10, 20, 30, 40), []int{10, 20, 30, 40}},
		{"full ring", buildQueue(3, 0, 10, 20, 30), []int{10, 20, 30}},
		{"wrapped", buildQueue(4, 3, 10, 20, 30), []int{10, 20, 30}},
		{"wrapped at the last cell", buildQueue(3, 2, 10, 20), []int{10, 20}},
		{"duplicates", queueOf(10, 10, 10), []int{10, 10, 10}},
		{"zero values", queueOf(0, 0), []int{0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.queue.ToSlice(); !slices.Equal(got, tc.want) {
				t.Errorf("ToSlice() = %v, want %v front to back", got, tc.want)
			}
			assertQueue(t, tc.queue, tc.want)
		})
	}

	t.Run("empty is non-nil", func(t *testing.T) {
		got := buildQueue[int](0, 0).ToSlice()
		if got == nil {
			t.Fatal("ToSlice() = nil, want an empty non-nil slice")
		}
		if len(got) != 0 {
			t.Errorf("ToSlice() = %v, want empty", got)
		}
	})

	// Returning the ring itself would pass the unwrapped cases above and hand the
	// caller the queue's own storage, in storage order rather than queue order.
	t.Run("result is a copy", func(t *testing.T) {
		q := queueOf(10, 20, 30)
		got := q.ToSlice()
		got[0] = 99
		assertQueue(t, q, []int{10, 20, 30})
	})
}

func TestQueueString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		queue *Queue[int]
		want  string
	}{
		{"unallocated", buildQueue[int](0, 0), "nil"},
		{"allocated and empty", buildQueue[int](4, 2), "nil"},
		{"one", queueOf(10), "10 -> nil"},
		{"two", queueOf(10, 20), "10 -> 20 -> nil"},
		{"many", queueOf(10, 20, 30), "10 -> 20 -> 30 -> nil"},
		{"wrapped", buildQueue(4, 3, 10, 20, 30), "10 -> 20 -> 30 -> nil"},
		{"zero value", queueOf(0), "0 -> nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.queue.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestQueuePrintQueue delegates to String by design - printing the rendered queue is
// the behavior under test - so it goes red if String is broken.
func TestQueuePrintQueue(t *testing.T) {
	for _, tc := range []struct {
		name  string
		queue *Queue[int]
		want  string
	}{
		{"empty", buildQueue[int](0, 0), "nil\n"},
		{"one", queueOf(10), "10 -> nil\n"},
		{"many", queueOf(10, 20, 30), "10 -> 20 -> 30 -> nil\n"},
		{"wrapped", buildQueue(4, 3, 10, 20, 30), "10 -> 20 -> 30 -> nil\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := captureStdout(t, tc.queue.PrintQueue)
			if got != tc.want {
				t.Errorf("PrintQueue() wrote %q, want %q", got, tc.want)
			}
		})
	}
}

// TestQueueCycles is the round trip the ring exists for: hold a few elements while
// pushing far more than the ring's length through it. A queue that reslices instead
// of wrapping passes every single-operation test above and grows without bound here.
// It uses Enqueue and Dequeue together on purpose, which is the one place in this
// file that is true.
func TestQueueCycles(t *testing.T) {
	q := buildQueue[int](0, 0)
	var want []int

	for i := 0; i < 200; i++ {
		q.Enqueue(i)
		want = append(want, i)
		if len(want) > 3 {
			got, err := q.Dequeue()
			if err != nil {
				t.Fatalf("Dequeue() at %d error = %v, want nil", i, err)
			}
			if got != want[0] {
				t.Fatalf("Dequeue() at %d = %v, want %v", i, got, want[0])
			}
			want = want[1:]
		}
		assertQueue(t, q, want)
	}

	if len(q.items) > 16 {
		t.Errorf("ring grew to %d cells to hold %d elements, want it to reuse the cells it frees", len(q.items), len(want))
	}
}

func ExampleQueue() {
	q := New[string]()
	q.Enqueue("a")
	q.Enqueue("b")
	q.Enqueue("c")
	q.PrintQueue()

	front, _ := q.Dequeue()
	fmt.Println(front, q.Len())
	// Output:
	// a -> b -> c -> nil
	// a 2
}
