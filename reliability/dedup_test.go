package reliability

import "testing"

func TestRecentWindow(t *testing.T) {
	w := NewRecentWindow(2)
	if !w.First(1) || w.First(1) {
		t.Fatal("duplicate was not rejected")
	}
	w.First(2)
	w.First(3)
	if !w.First(1) {
		t.Fatal("old sequence must leave the bounded window")
	}
}
