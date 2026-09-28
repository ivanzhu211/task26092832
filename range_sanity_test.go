package cellprotect

import "testing"

func TestCellRangeSanity(t *testing.T) {
	// 2x2 all suppressed, margins R=(6,4) C=(6,4): x00 in [2,4]... check:
	// min = max(0, 6+6-10)=2, max = min(6,6)=6
	sup := [][]bool{{true, true}, {true, true}}
	lo, hi := cellRange(2, 2, []int{6, 4}, []int{6, 4}, sup, 0, 0)
	if lo != 2 || hi != 6 {
		t.Fatalf("got [%d,%d], want [2,6]", lo, hi)
	}
	// pinned cell: single suppressed cell in its component
	sup2 := [][]bool{{true, false}, {false, false}}
	lo, hi = cellRange(2, 2, []int{3, 0}, []int{3, 0}, sup2, 0, 0)
	if lo != 3 || hi != 3 {
		t.Fatalf("got [%d,%d], want [3,3]", lo, hi)
	}
}
