package cellprotect_test

import (
	"math/rand"
	"testing"

	cp "cellprotect"
)

// bruteRange enumerates every non-negative integer table consistent with the
// published cells and totals, and returns the exact min/max of the queried
// cell. Used only on tiny tables (total <= 12).
func bruteRange(p cp.PublishedReport, row, col int) (lo, hi int) {
	m := len(p.Cells)
	n := len(p.Cells[0])

	grid := make([][]int, m)
	rowUsed := make([]int, m)
	colUsed := make([]int, n)
	supp := make([][2]int, 0, m*n)
	for i := 0; i < m; i++ {
		grid[i] = make([]int, n)
		for j := 0; j < n; j++ {
			if p.Cells[i][j].Kind == cp.CellPublished {
				v := p.Cells[i][j].Value
				grid[i][j] = v
				rowUsed[i] += v
				colUsed[j] += v
			} else {
				supp = append(supp, [2]int{i, j})
			}
		}
	}

	lo, hi = -1, -1
	var dfs func(k int)
	dfs = func(k int) {
		if k == len(supp) {
			for j := 0; j < n; j++ {
				if colUsed[j] != p.ColTotals[j] {
					return
				}
			}
			v := grid[row][col]
			if lo < 0 || v < lo {
				lo = v
			}
			if v > hi {
				hi = v
			}
			return
		}
		i, j := supp[k][0], supp[k][1]
		room := p.RowTotals[i] - rowUsed[i]
		for x := 0; x <= room; x++ {
			if colUsed[j]+x > p.ColTotals[j] {
				break
			}
			grid[i][j] = x
			rowUsed[i] += x
			colUsed[j] += x
			dfs(k + 1)
			rowUsed[i] -= x
			colUsed[j] -= x
		}
		grid[i][j] = 0
	}
	_ = rowUsed
	dfs(0)
	return lo, hi
}

func allSuppressedReport(c cp.Counts) cp.PublishedReport {
	m, n := len(c.Cells), len(c.Cols)
	p := cp.PublishedReport{
		Rows:      c.Rows,
		Cols:      c.Cols,
		Cells:     make([][]cp.PublishedCell, m),
		RowTotals: append([]int(nil), c.RowTotals...),
		ColTotals: append([]int(nil), c.ColTotals...),
		Total:     c.Total,
	}
	for i := 0; i < m; i++ {
		p.Cells[i] = make([]cp.PublishedCell, n)
		for j := 0; j < n; j++ {
			p.Cells[i][j] = cp.PublishedCell{Value: c.Cells[i][j], Kind: cp.CellSuppressed}
		}
	}
	return p
}

// protectedByAPI checks the protection standard for every primary cell using
// only the public FeasibleRange.
func protectedByAPI(t *testing.T, c cp.Counts, p cp.PublishedReport, primaries [][2]int) bool {
	t.Helper()
	ok := true
	for _, e := range primaries {
		lo, hi, err := cp.FeasibleRange(p, e[0], e[1])
		if err != nil {
			t.Fatalf("primary cell not suppressed: %v", err)
		}
		if !(lo == 0 || hi >= cp.K) {
			ok = false
		}
	}
	return ok
}

func primaryCells(c cp.Counts) [][2]int {
	var out [][2]int
	for i := range c.Cells {
		for j := range c.Cells[i] {
			if v := c.Cells[i][j]; v >= 1 && v <= cp.K-1 {
				out = append(out, [2]int{i, j})
			}
		}
	}
	return out
}

// verifyMinimality unpublishes each complementary cell one at a time and
// requires at least one primary cell to become unprotected.
func verifyMinimality(t *testing.T, c cp.Counts, internal cp.InternalReport) {
	t.Helper()
	m, n := len(c.Cells), len(c.Cols)
	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			if internal.Suppression[i][j] != cp.Complementary {
				continue
			}
			trial := allSuppressedReport(c)
			for a := 0; a < m; a++ {
				for b := 0; b < n; b++ {
					if internal.Suppression[a][b] == cp.Published || (a == i && b == j) {
						trial.Cells[a][b].Kind = cp.CellPublished
					}
				}
			}
			necessary := false
			for _, e := range primaryCells(c) {
				lo, hi, err := cp.FeasibleRange(trial, e[0], e[1])
				if err == nil && lo >= 1 && hi <= cp.K-1 {
					necessary = true
					break
				}
			}
			if !necessary {
				t.Fatalf("complementary suppression (%d,%d) is not necessary", i, j)
			}
		}
	}
}

func randCounts(rng *rand.Rand, m, n, total int) cp.Counts {
	cells := make([][]int, m)
	rt := make([]int, m)
	ct := make([]int, n)
	for i := range cells {
		cells[i] = make([]int, n)
	}
	for b := 0; b < total; b++ {
		cell := rng.Intn(m * n)
		i, j := cell/n, cell%n
		cells[i][j]++
		rt[i]++
		ct[j]++
	}
	rows := make([]string, m)
	cols := make([]string, n)
	for i := range rows {
		rows[i] = string(rune('A' + i))
	}
	for j := range cols {
		cols[j] = string(rune('1' + j))
	}
	return cp.Counts{Rows: rows, Cols: cols, Cells: cells, RowTotals: rt, ColTotals: ct, Total: total}
}

func TestRandomSmallTables(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	const iterations = 300
	for it := 0; it < iterations; it++ {
		m := 1 + rng.Intn(4)
		n := 1 + rng.Intn(4)
		total := rng.Intn(13)
		c := randCounts(rng, m, n, total)
		primaries := primaryCells(c)

		internal := cp.Protect(c)

		if internal.Rejected {
			// Verification 3: with all cells suppressed some primary still
			// violates the standard.
			if len(primaries) == 0 {
				t.Fatalf("rejected table with no primary cells")
			}
			allSupp := allSuppressedReport(c)
			bad := 0
			for _, e := range primaries {
				lo, hi, err := cp.FeasibleRange(allSupp, e[0], e[1])
				if err != nil {
					t.Fatal(err)
				}
				if lo >= 1 && hi <= cp.K-1 {
					bad++
				}
			}
			if bad == 0 {
				t.Fatalf("rejected but full suppression protects every primary:\n%v", c.Cells)
			}
			continue
		}

		pub := cp.PublishedReport{
			Rows:      internal.Rows,
			Cols:      internal.Cols,
			Cells:     make([][]cp.PublishedCell, m),
			RowTotals: internal.RowTotals,
			ColTotals: internal.ColTotals,
			Total:     internal.Total,
		}
		for i := 0; i < m; i++ {
			pub.Cells[i] = make([]cp.PublishedCell, n)
			for j := 0; j < n; j++ {
				cell := cp.PublishedCell{Value: c.Cells[i][j]}
				if internal.Suppression[i][j] == cp.Published {
					cell.Kind = cp.CellPublished
				} else {
					cell.Kind = cp.CellSuppressed
				}
				pub.Cells[i][j] = cell
			}
		}

		// Verification 1: protection standard + minimality.
		if !protectedByAPI(t, c, pub, primaries) {
			t.Fatalf("primary not protected:\ncells=%v\nkinds=%v", c.Cells, internal.Suppression)
		}
		verifyMinimality(t, c, internal)

		// Verification 2: FeasibleRange matches brute force for every
		// suppressed cell.
		for i := 0; i < m; i++ {
			for j := 0; j < n; j++ {
				if pub.Cells[i][j].Kind != cp.CellSuppressed {
					continue
				}
				lo, hi, err := cp.FeasibleRange(pub, i, j)
				if err != nil {
					t.Fatal(err)
				}
				blo, bhi := bruteRange(pub, i, j)
				if blo < 0 {
					t.Fatalf("brute enumeration found no feasible table")
				}
				if lo != blo || hi != bhi {
					t.Fatalf("range mismatch at (%d,%d): flow=[%d,%d] brute=[%d,%d]\ncells=%v",
						i, j, lo, hi, blo, bhi, c.Cells)
				}
			}
		}
	}
}

// TestManualCycleTable: suppressing exactly two cells per row and per column
// (the 6-cycle) still leaves every primary cell pinned inside 1..4 via the
// totals, even though every row/column has two suppressions. A seventh edge
// is required.
func TestManualCycleTable(t *testing.T) {
	c := cp.Counts{
		Rows: []string{"D1", "D2", "D3"},
		Cols: []string{"c1", "c2", "c3"},
		Cells: [][]int{
			{1, 3, 4},
			{3, 1, 2},
			{4, 2, 0},
		},
		RowTotals: []int{8, 6, 6},
		ColTotals: []int{8, 6, 6},
		Total:     20,
	}

	// Demonstrate the 2-per-row/2-per-column rule is insufficient:
	// suppress the 6-cycle and every primary remains in 1..4.
	cycle := allSuppressedReport(c)
	cycleMask := [6][2]int{{0, 0}, {0, 1}, {1, 1}, {1, 2}, {2, 0}, {2, 2}}
	cycleSet := map[[2]int]bool{}
	for _, e := range cycleMask {
		cycleSet[e] = true
	}
	for i := range c.Cells {
		for j := range c.Cells[i] {
			if !cycleSet[[2]int{i, j}] {
				cycle.Cells[i][j].Kind = cp.CellPublished
			}
		}
	}
	for _, e := range [][2]int{{0, 0}, {0, 1}, {1, 1}, {2, 0}} {
		lo, hi, err := cp.FeasibleRange(cycle, e[0], e[1])
		if err != nil {
			t.Fatal(err)
		}
		if !(lo >= 1 && hi <= 4) {
			t.Fatalf("cycle cell (%d,%d) expected pinned to 1..4, got [%d,%d]", e[0], e[1], lo, hi)
		}
	}

	internal := cp.Protect(c)
	if internal.Rejected {
		t.Fatalf("should be publishable: %s", internal.RejectReason)
	}
	pub := toPublished(c, internal)
	for _, e := range primaryCells(c) {
		lo, hi, err := cp.FeasibleRange(pub, e[0], e[1])
		if err != nil {
			t.Fatal(err)
		}
		if !(lo == 0 || hi >= cp.K) {
			t.Fatalf("primary (%d,%d) unprotected: [%d,%d]", e[0], e[1], lo, hi)
		}
	}
	verifyMinimality(t, c, internal)
}

// TestManualZeroColumnTable: an all-zero column offers the most "convenient"
// extra suppression slot, but a cell in a total-0 column never relaxes a
// primary range. Protect must choose a useful cell in a non-zero column.
func TestManualZeroColumnTable(t *testing.T) {
	c := cp.Counts{
		Rows: []string{"D1", "D2", "D3"},
		Cols: []string{"c1", "c2", "c3", "z"},
		Cells: [][]int{
			{1, 3, 4, 0},
			{3, 1, 2, 0},
			{4, 2, 0, 0},
		},
		RowTotals: []int{8, 6, 6},
		ColTotals: []int{8, 6, 6, 0},
		Total:     20,
	}

	// Trap: 6-cycle suppressions plus a cell in the zero-total column.
	// Every primary is still pinned inside 1..4.
	trap := allSuppressedReport(c)
	trapSet := map[[2]int]bool{
		{0, 0}: true, {0, 1}: true, {1, 1}: true,
		{1, 2}: true, {2, 0}: true, {2, 2}: true, {0, 3}: true,
	}
	for i := range c.Cells {
		for j := range c.Cells[i] {
			if !trapSet[[2]int{i, j}] {
				trap.Cells[i][j].Kind = cp.CellPublished
			}
		}
	}
	lo, hi, err := cp.FeasibleRange(trap, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !(lo >= 1 && hi <= 4) {
		t.Fatalf("zero-column suppression unexpectedly helps: [%d,%d]", lo, hi)
	}

	internal := cp.Protect(c)
	if internal.Rejected {
		t.Fatalf("should be publishable: %s", internal.RejectReason)
	}
	for i := range internal.Suppression {
		if internal.Suppression[i][3] != cp.Published {
			t.Fatalf("cell in zero-total column was suppressed")
		}
	}
	pub := toPublished(c, internal)
	for _, e := range primaryCells(c) {
		lo, hi, err := cp.FeasibleRange(pub, e[0], e[1])
		if err != nil {
			t.Fatal(err)
		}
		if !(lo == 0 || hi >= cp.K) {
			t.Fatalf("primary (%d,%d) unprotected: [%d,%d]", e[0], e[1], lo, hi)
		}
	}
	verifyMinimality(t, c, internal)
}

func toPublished(c cp.Counts, internal cp.InternalReport) cp.PublishedReport {
	m, n := len(c.Cells), len(c.Cols)
	p := cp.PublishedReport{
		Rows:      internal.Rows,
		Cols:      internal.Cols,
		Cells:     make([][]cp.PublishedCell, m),
		RowTotals: internal.RowTotals,
		ColTotals: internal.ColTotals,
		Total:     internal.Total,
	}
	for i := 0; i < m; i++ {
		p.Cells[i] = make([]cp.PublishedCell, n)
		for j := 0; j < n; j++ {
			cell := cp.PublishedCell{Value: c.Cells[i][j]}
			if internal.Suppression[i][j] == cp.Published {
				cell.Kind = cp.CellPublished
			} else {
				cell.Kind = cp.CellSuppressed
			}
			p.Cells[i][j] = cell
		}
	}
	return p
}

func newRng(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }
