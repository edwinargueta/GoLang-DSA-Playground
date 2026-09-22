package hashtable

// slotState separates a cell that has never held anything from one whose entry was
// deleted. A probe has to treat them differently: an empty cell ends a search
// because nothing could have been placed beyond it, while a tombstone must be
// probed straight through.
type slotState uint8

const (
	slotEmpty slotState = iota
	slotFilled
	slotDeleted
)

// slot is one cell of the open-addressed table: a key, its value, and which of the
// three states the cell is in.
type slot[K comparable, V any] struct {
	key   K
	value V
	state slotState
}
