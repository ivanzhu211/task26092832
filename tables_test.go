package cellprotect

import "testing"

// makeCounts builds a count table from row-major values.
func makeCounts(cells ...[]int) *Counts {
	counts := &Counts{Cells: cells}
	for r := range cells {
		counts.Rows = append(counts.Rows, []string{"A", "B", "C", "D"}[r])
	}
	for c := range cells[0] {
		counts.Cols = append(counts.Cols, []string{"v0", "v1", "v2", "v3"}[c])
	}
	return counts
}

// naiveView builds the published form produced by the naive
// "at least two suppressed cells per row and column" rule: suppress the
// given cells, publish everything else.
func naiveView(counts *Counts, suppressed ...cellPos) *PublishedTable {
	pub := allSuppressedView(counts)
	sup := make(map[cellPos]bool)
	for _, p := range suppressed {
		sup[p] = true
	}
	for r := range pub.Cells {
		for c := range pub.Cells[r] {
			if !sup[cellPos{r, c}] {
				pub.Cells[r][c] = PublishedCell{Value: counts.Cells[r][c]}
			}
		}
	}
	return pub
}

// assertStandard verifies via FeasibleRange that every primary cell of the
// protection result satisfies the protection standard.
func assertStandard(t *testing.T, prot *Protection) {
	t.Helper()
	if prot.Rejected {
		t.Fatalf("table unexpectedly rejected: %s", prot.Reason)
	}
	pub := prot.Published()
	for r := range prot.State {
		for c, st := range prot.State[r] {
			if st != StatePrimary {
				continue
			}
			lo, hi := FeasibleRange(pub, r, c)
			if lo >= 1 && hi <= k-1 {
				t.Fatalf("primary cell (%d,%d) feasible range [%d,%d] violates standard", r, c, lo, hi)
			}
		}
	}
}

// Table 1: the naive "two suppressed cells per row and column" rule leaves
// a primary cell pinned into [1,4] by the totals.
//
//	 2  1  7 | 10
//	 1  1  8 | 10
//	 7  8  5 | 20
//	10 10 20 | 40
//
// The four primary cells already give every affected row and column two
// suppressed cells, yet cell (0,0) is then forced into [1,3].
func TestNaiveTwoPerRowColIsNotEnough(t *testing.T) {
	counts := makeCounts(
		[]int{2, 1, 7},
		[]int{1, 1, 8},
		[]int{7, 8, 5},
	)
	naive := naiveView(counts, cellPos{0, 0}, cellPos{0, 1}, cellPos{1, 0}, cellPos{1, 1})
	lo, hi := FeasibleRange(naive, 0, 0)
	if lo != 1 || hi != 3 {
		t.Fatalf("naive pattern: cell (0,0) range = [%d,%d], want [1,3]", lo, hi)
	}
	// The naive rule is satisfied (two suppressed cells in each of rows
	// 0-1 and columns 0-1) but the range [1,3] lies entirely in [1,4].
	prot := Protect(counts)
	assertStandard(t, prot)
}

// Table 2: the most convenient complementary cell lies in a zero-total
// column, where suppressing it cannot help the primary cell at all.
//
//	 2  8  0 | 10
//	 8  2  0 | 10
//	10 10  0 | 20
//
// Suppressing the zero cell (0,2) next to primary cell (0,0) leaves (0,0)
// pinned at exactly 2, because the zero column total gives nothing away.
func TestZeroTotalColumnSuppressionIsUseless(t *testing.T) {
	counts := makeCounts(
		[]int{2, 8, 0},
		[]int{8, 2, 0},
	)
	naive := naiveView(counts, cellPos{0, 0}, cellPos{1, 1}, cellPos{0, 2})
	lo, hi := FeasibleRange(naive, 0, 0)
	if lo != 2 || hi != 2 {
		t.Fatalf("suppressing zero-column cell: cell (0,0) range = [%d,%d], want [2,2]", lo, hi)
	}
	prot := Protect(counts)
	assertStandard(t, prot)
	// The useless zero-column cells must not be suppressed in the output:
	// they could never be necessary.
	for _, p := range []cellPos{{0, 2}, {1, 2}} {
		if prot.State[p.r][p.c] != StatePublished {
			t.Fatalf("zero-total column cell (%d,%d) suppressed although it cannot help", p.r, p.c)
		}
	}
}
