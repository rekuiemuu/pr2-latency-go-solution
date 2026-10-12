package reliability

// RecentWindow remembers a bounded number of processed sequence numbers.
type RecentWindow struct {
	limit int
	seen  map[uint16]bool
	order []uint16
}

func NewRecentWindow(limit int) *RecentWindow {
	if limit < 1 {
		limit = 1
	}
	return &RecentWindow{limit: limit, seen: make(map[uint16]bool)}
}

// First returns true once for a sequence and false for its duplicates.
func (w *RecentWindow) First(sequence uint16) bool {
	if w.seen[sequence] {
		return false
	}
	w.seen[sequence] = true
	w.order = append(w.order, sequence)
	if len(w.order) > w.limit {
		delete(w.seen, w.order[0])
		w.order = w.order[1:]
	}
	return true
}
