package cellprotect_test

import (
	"fmt"
	"math/rand"
	"testing"

	cp "cellprotect"
)

// makePerfCounts builds a 40x8 table with roughly target primary cells.
func makePerfCounts(rng *rand.Rand, targetPrimary int) cp.Counts {
	m, n := 40, 8
	cells := make([][]int, m)
	rt := make([]int, m)
	ct := make([]int, n)
	primary := 0
	for i := range cells {
		cells[i] = make([]int, n)
		for j := 0; j < n; j++ {
			var v int
			r := rng.Float64()
			if primary < targetPrimary && r < 0.30 {
				v = 1 + rng.Intn(4) // 1..4 primary
				primary++
			} else if r < 0.55 {
				v = 5 + rng.Intn(20) // >=5 safe
			} else if r < 0.8 {
				v = 0
			} else {
				v = 5 + rng.Intn(40)
			}
			cells[i][j] = v
			rt[i] += v
			ct[j] += v
		}
	}
	rows := make([]string, m)
	cols := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf("D%02d", i)
	}
	for j := range cols {
		cols[j] = fmt.Sprintf("c%d", j)
	}
	total := 0
	for _, r := range rt {
		total += r
	}
	return cp.Counts{Rows: rows, Cols: cols, Cells: cells, RowTotals: rt, ColTotals: ct, Total: total}
}

func TestPerformance40x8(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	rng := rand.New(rand.NewSource(123))
	c := makePerfCounts(rng, 80)
	if got := len(primaryCells(c)); got < 60 {
		t.Fatalf("only %d primary cells, want ~80", got)
	}
	internal := cp.Protect(c)
	if internal.Rejected {
		t.Skipf("table rejected (random): %s", internal.RejectReason)
	}
	pub := toPublished(c, internal)
	primaries := primaryCells(c)
	for _, e := range primaries {
		lo, hi, err := cp.FeasibleRange(pub, e[0], e[1])
		if err != nil {
			t.Fatal(err)
		}
		if !(lo == 0 || hi >= cp.K) {
			t.Fatalf("primary (%d,%d) unprotected [%d,%d]", e[0], e[1], lo, hi)
		}
	}
}

func BenchmarkProtect40x8(b *testing.B) {
	rng := rand.New(rand.NewSource(123))
	c := makePerfCounts(rng, 80)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cp.Protect(c)
	}
}

func TestPerformanceAdversarial(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	m, n := 40, 8
	cells := make([][]int, m)
	rt := make([]int, m)
	ct := make([]int, n)
	rng := rand.New(rand.NewSource(999))
	for i := range cells {
		cells[i] = make([]int, n)
		for j := 0; j < n; j++ {
			v := 1 + rng.Intn(4)
			cells[i][j] = v
			rt[i] += v
			ct[j] += v
		}
	}
	rows := make([]string, m)
	cols := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf("D%02d", i)
	}
	for j := range cols {
		cols[j] = fmt.Sprintf("c%d", j)
	}
	total := 0
	for _, r := range rt {
		total += r
	}
	c := cp.Counts{Rows: rows, Cols: cols, Cells: cells, RowTotals: rt, ColTotals: ct, Total: total}
	if len(primaryCells(c)) != m*n {
		t.Fatal("expected every cell primary")
	}
	internal := cp.Protect(c)
	if internal.Rejected {
		t.Fatalf("should be publishable (all suppressed trivially protects): %s", internal.RejectReason)
	}
	pub := toPublished(c, internal)
	for _, e := range primaryCells(c) {
		lo, hi, err := cp.FeasibleRange(pub, e[0], e[1])
		if err != nil {
			t.Fatal(err)
		}
		if !(lo == 0 || hi >= cp.K) {
			t.Fatalf("primary (%d,%d) unprotected [%d,%d]", e[0], e[1], lo, hi)
		}
	}
}

// TestPerformanceGreedyStress forces many greedy rounds: 20 separate 2x2
// primary blocks (40 primaries) where each block requires a complementary
// suppression from its own row group's candidate cells.
func TestPerformanceGreedyStress(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	m, n := 40, 8
	cells := make([][]int, m)
	rt := make([]int, m)
	ct := make([]int, n)
	for i := range cells {
		cells[i] = make([]int, n)
	}
	for g := 0; g < 20; g++ {
		r0 := g * 2
		// Primary 2x2 block in columns 0,1: values in 1..4 arranged so the
		// block alone pins each primary inside 1..4.
		block := [2][2]int{{2, 3}, {3, 2}}
		for a := 0; a < 2; a++ {
			for b := 0; b < 2; b++ {
				v := block[a][b]
				cells[r0+a][b] = v
				rt[r0+a] += v
				ct[b] += v
			}
		}
		// Candidate cells in columns 2..7, large values that can serve as
		// complementary suppressions (one per row pair needed).
		for col := 2; col < 8; col++ {
			v := 7 + (r0 % 10) + col
			cells[r0][col] = v
			rt[r0] += v
			ct[col] += v
			cells[r0+1][col] = v
			rt[r0+1] += v
			ct[col] += v
		}
	}
	rows := make([]string, m)
	cols := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf("D%02d", i)
	}
	for j := range cols {
		cols[j] = fmt.Sprintf("c%d", j)
	}
	total := 0
	for _, r := range rt {
		total += r
	}
	c := cp.Counts{Rows: rows, Cols: cols, Cells: cells, RowTotals: rt, ColTotals: ct, Total: total}
	if nprim := len(primaryCells(c)); nprim != 80 {
		t.Fatalf("got %d primaries, want 80", nprim)
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
			t.Fatalf("primary (%d,%d) unprotected [%d,%d]", e[0], e[1], lo, hi)
		}
	}
}

// TestPerformanceWorstCase constructs 40 independent 2x2 components using
// 8 unique columns and 40 rows (components placed on disjoint row pairs and
// column pairs, sharing only fillers), maximizing greedy rounds.
func TestPerformanceWorstCase(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	m, n := 40, 8
	cells := make([][]int, m)
	rt := make([]int, m)
	ct := make([]int, n)
	rng := rand.New(rand.NewSource(55))
	for i := range cells {
		cells[i] = make([]int, n)
	}
	// With only 8 columns we can have at most 4 disjoint column pairs; place
	// 10 components per column pair across disjoint row pairs = 40 rows.
	for g := 0; g < 40; g++ {
		r0 := g
		c0 := (g % 4) * 2
		block := [2][2]int{{2, 3}, {3, 2}}
		for b := 0; b < 2; b++ {
			v := block[0][b]
			cells[r0][c0+b] = v
			rt[r0] += v
			ct[c0+b] += v
		}
		// fill remaining primary cells of other column pairs with large vals
		for col := 0; col < 8; col++ {
			if col >= c0 && col < c0+2 {
				continue
			}
			v := 8 + rng.Intn(30)
			cells[r0][col] = v
			rt[r0] += v
			ct[col] += v
		}
	}
	rows := make([]string, m)
	cols := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf("D%02d", i)
	}
	for j := range cols {
		cols[j] = fmt.Sprintf("c%d", j)
	}
	total := 0
	for _, r := range rt {
		total += r
	}
	c := cp.Counts{Rows: rows, Cols: cols, Cells: cells, RowTotals: rt, ColTotals: ct, Total: total}
	t.Logf("primary count: %d", len(primaryCells(c)))
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
			t.Fatalf("primary (%d,%d) unprotected [%d,%d]", e[0], e[1], lo, hi)
		}
	}
}

// TestPerformance20IsolatedBlocks: 20 flow-isolated 2x2 primary blocks in
// columns 0..3 across 40 rows; columns 4..7 are filler-only. Each block must
// independently get a complementary suppression, forcing ~20 greedy rounds.
func TestPerformance20IsolatedBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	m, n := 40, 8
	cells := make([][]int, m)
	rt := make([]int, m)
	ct := make([]int, n)
	for i := range cells {
		cells[i] = make([]int, n)
	}
	for g := 0; g < 20; g++ {
		r0 := g * 2
		c0 := (g % 2) * 2
		block := [2][2]int{{2, 3}, {3, 2}}
		for a := 0; a < 2; a++ {
			for b := 0; b < 2; b++ {
				v := block[a][b]
				cells[r0+a][c0+b] = v
				rt[r0+a] += v
				ct[c0+b] += v
			}
		}
	}
	// Filler-only columns 4..7: large values on every row (candidate edges).
	for i := 0; i < m; i++ {
		for col := 4; col < 8; col++ {
			v := 9 + ((i*7 + col*3) % 20)
			cells[i][col] = v
			rt[i] += v
			ct[col] += v
		}
	}
	rows := make([]string, m)
	cols := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf("D%02d", i)
	}
	for j := range cols {
		cols[j] = fmt.Sprintf("c%d", j)
	}
	total := 0
	for _, r := range rt {
		total += r
	}
	c := cp.Counts{Rows: rows, Cols: cols, Cells: cells, RowTotals: rt, ColTotals: ct, Total: total}
	if nprim := len(primaryCells(c)); nprim != 80 {
		t.Fatalf("got %d primaries, want 80", nprim)
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
			t.Fatalf("primary (%d,%d) unprotected [%d,%d]", e[0], e[1], lo, hi)
		}
	}
}

// TestPerformance4DisjointBlocks: four truly disjoint 2x2 primary blocks
// (8 distinct rows, 8 distinct cols); each block is flow-isolated and needs
// its own complementary suppression, forcing several greedy rounds.
func TestPerformance4DisjointBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	m, n := 40, 8
	cells := make([][]int, m)
	rt := make([]int, m)
	ct := make([]int, n)
	for i := range cells {
		cells[i] = make([]int, n)
	}
	// 4 blocks: rows 0-7, each block uses its own 2 rows and 2 columns.
	for b := 0; b < 4; b++ {
		r0 := b * 2
		c0 := b * 2
		block := [2][2]int{{2, 3}, {3, 2}}
		for a := 0; a < 2; a++ {
			for cc := 0; cc < 2; cc++ {
				v := block[a][cc]
				cells[r0+a][c0+cc] = v
				rt[r0+a] += v
				ct[c0+cc] += v
			}
		}
	}
	// Every block row already has values in its 2 columns; add no edges on
	// other primary columns. Filler rows 8..39 carry large values in all 8
	// columns to provide residual capacity and candidate edges.
	for i := 8; i < m; i++ {
		for j := 0; j < n; j++ {
			v := 10 + ((i*13 + j*7) % 30)
			cells[i][j] = v
			rt[i] += v
			ct[j] += v
		}
	}
	// Block rows have edges ONLY in their own two columns (cells in other
	// columns stay 0 and unpublished... they remain 0 published, so no
	// candidate edges there). Complementary slots therefore come from the
	// filler rows of the block's two columns.
	rows := make([]string, m)
	cols := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf("D%02d", i)
	}
	for j := range cols {
		cols[j] = fmt.Sprintf("c%d", j)
	}
	total := 0
	for _, r := range rt {
		total += r
	}
	c := cp.Counts{Rows: rows, Cols: cols, Cells: cells, RowTotals: rt, ColTotals: ct, Total: total}
	if nprim := len(primaryCells(c)); nprim != 16 {
		t.Fatalf("got %d primaries, want 16", nprim)
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
			t.Fatalf("primary (%d,%d) unprotected [%d,%d]", e[0], e[1], lo, hi)
		}
	}
}

// TestPerformanceSparseRowComponents builds 20 flow-isolated 2x8 components:
// a dense primary row (7 primaries) plus a second row with one primary and
// six large published candidate cells. Each component needs about five
// complementary suppressions, maximizing greedy rounds at 40x8.
func TestPerformanceSparseRowComponents(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	m, n := 40, 8
	cells := make([][]int, m)
	rt := make([]int, m)
	ct := make([]int, n)
	for i := range cells {
		cells[i] = make([]int, n)
	}
	for g := 0; g < 20; g++ {
		r0 := g * 2
		for j := 0; j < 7; j++ {
			v := 2
			cells[r0][j] = v
			rt[r0] += v
			ct[j] += v
		}
		cells[r0][7] = 0
		// second row: primary at col 0, large published candidates at 1..7
		cells[r0+1][0] = 1
		rt[r0+1] += 1
		ct[0] += 1
		for j := 1; j < 8; j++ {
			v := 20
			cells[r0+1][j] = v
			rt[r0+1] += v
			ct[j] += v
		}
	}
	rows := make([]string, m)
	cols := make([]string, n)
	for i := range rows {
		rows[i] = fmt.Sprintf("D%02d", i)
	}
	for j := range cols {
		cols[j] = fmt.Sprintf("c%d", j)
	}
	total := 0
	for _, r := range rt {
		total += r
	}
	c := cp.Counts{Rows: rows, Cols: cols, Cells: cells, RowTotals: rt, ColTotals: ct, Total: total}
	if nprim := len(primaryCells(c)); nprim != 160 {
		t.Fatalf("got %d primaries, want 160", nprim)
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
			t.Fatalf("primary (%d,%d) unprotected [%d,%d]", e[0], e[1], lo, hi)
		}
	}
}
