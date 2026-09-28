package cellprotect

import "errors"

// ErrNotSuppressed is returned when FeasibleRange is asked about a cell that
// is not suppressed in the published report.
var ErrNotSuppressed = errors.New("cell is not suppressed")

// FeasibleRange computes the minimum and maximum integer value the
// suppressed cell (row, col) can take, consistent with every published cell
// and every published total (non-negative integer cells). It operates only
// on the releasable report and never enumerates possible tables.
func FeasibleRange(p PublishedReport, row, col int) (min, max int, err error) {
	m := len(p.Cells)
	n := 0
	if m > 0 {
		n = len(p.Cells[0])
	}
	if row < 0 || row >= m || col < 0 || col >= n {
		return 0, 0, ErrNotSuppressed
	}
	if p.Cells[row][col].Kind != CellSuppressed {
		return 0, 0, ErrNotSuppressed
	}

	net := flowNetwork{
		rows:  m,
		cols:  n,
		rRes:  make([]int64, m),
		cRes:  make([]int64, n),
		edges: make([][2]int, 0, m*n),
	}
	for i := 0; i < m; i++ {
		var rr int64 = int64(p.RowTotals[i])
		for j := 0; j < n; j++ {
			if p.Cells[i][j].Kind == CellSuppressed {
				net.edges = append(net.edges, [2]int{i, j})
			} else {
				rr -= int64(p.Cells[i][j].Value)
			}
		}
		net.rRes[i] = rr
	}
	for j := 0; j < n; j++ {
		var cc int64 = int64(p.ColTotals[j])
		for i := 0; i < m; i++ {
			if p.Cells[i][j].Kind != CellSuppressed {
				cc -= int64(p.Cells[i][j].Value)
			}
		}
		net.cRes[j] = cc
	}
	for _, rr := range net.rRes {
		net.total += rr
	}

	// Find the network edge index for (row, col).
	idx := -1
	for k, e := range net.edges {
		if e[0] == row && e[1] == col {
			idx = k
			break
		}
	}
	if idx < 0 {
		return 0, 0, ErrNotSuppressed
	}
	lo, hi := net.edgeRange(idx)
	return int(lo), int(hi), nil
}
