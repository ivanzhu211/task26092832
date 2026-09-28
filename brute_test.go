package cellprotect

// Brute-force enumerator used only in tests: enumerates every table of
// non-negative integers consistent with a published form (fixed published
// cells plus all margins) and derives exact feasible ranges.

type cellPos struct{ r, c int }

// bruteForceRanges returns the exact [min, max] feasible range of every
// suppressed cell in the published table.
func bruteForceRanges(pub *PublishedTable) map[cellPos][2]int {
	nRows := len(pub.Rows)
	nCols := len(pub.Cols)
	rowRem := make([]int, nRows)
	colRem := make([]int, nCols)
	var cells []cellPos
	for r := 0; r < nRows; r++ {
		rowRem[r] = pub.RowTotals[r]
	}
	for c := 0; c < nCols; c++ {
		colRem[c] = pub.ColTotals[c]
	}
	for r := 0; r < nRows; r++ {
		for c := 0; c < nCols; c++ {
			cell := pub.Cells[r][c]
			if cell.Suppressed {
				cells = append(cells, cellPos{r, c})
			} else {
				rowRem[r] -= cell.Value
				colRem[c] -= cell.Value
			}
		}
	}
	ranges := make(map[cellPos][2]int)
	for _, p := range cells {
		ranges[p] = [2]int{-1, -1}
	}
	values := make([]int, len(cells))
	var rec func(i int)
	rec = func(i int) {
		if i == len(cells) {
			for j, p := range cells {
				rng := ranges[p]
				if rng[0] == -1 || values[j] < rng[0] {
					rng[0] = values[j]
				}
				if rng[1] == -1 || values[j] > rng[1] {
					rng[1] = values[j]
				}
				ranges[p] = rng
			}
			return
		}
		p := cells[i]
		upper := rowRem[p.r]
		if colRem[p.c] < upper {
			upper = colRem[p.c]
		}
		for v := 0; v <= upper; v++ {
			values[i] = v
			rowRem[p.r] -= v
			colRem[p.c] -= v
			// Prune: remaining residuals must stay reachable with the
			// cells still unassigned.
			if feasibleResidual(cells, i+1, rowRem, colRem) {
				rec(i + 1)
			}
			rowRem[p.r] += v
			colRem[p.c] += v
		}
	}
	rec(0)
	return ranges
}

// feasibleResidual checks that remaining residuals can still be satisfied
// by the unassigned cells (necessary conditions only, for pruning).
func feasibleResidual(cells []cellPos, start int, rowRem, colRem []int) bool {
	for _, v := range rowRem {
		if v < 0 {
			return false
		}
	}
	for _, v := range colRem {
		if v < 0 {
			return false
		}
	}
	rowCap := make([]int, len(rowRem))
	colCap := make([]int, len(colRem))
	for i := start; i < len(cells); i++ {
		rowCap[cells[i].r]++
		colCap[cells[i].c]++
	}
	for r, v := range rowRem {
		if rowCap[r] == 0 && v != 0 {
			return false
		}
	}
	for c, v := range colRem {
		if colCap[c] == 0 && v != 0 {
			return false
		}
	}
	return true
}

// allSuppressedView builds the published form in which every cell is
// suppressed and only the margins are known.
func allSuppressedView(counts *Counts) *PublishedTable {
	nRows := len(counts.Rows)
	nCols := len(counts.Cols)
	pub := &PublishedTable{
		Rows:      counts.Rows,
		Cols:      counts.Cols,
		Cells:     make([][]PublishedCell, nRows),
		RowTotals: make([]int, nRows),
		ColTotals: make([]int, nCols),
		Total:     counts.Total(),
	}
	for r := 0; r < nRows; r++ {
		pub.Cells[r] = make([]PublishedCell, nCols)
		for c := 0; c < nCols; c++ {
			pub.Cells[r][c] = PublishedCell{Suppressed: true}
		}
		pub.RowTotals[r] = counts.RowTotal(r)
	}
	for c := 0; c < nCols; c++ {
		pub.ColTotals[c] = counts.ColTotal(c)
	}
	return pub
}

// viewWithExtraPublished returns the published form of a protection result
// with the given cell forced back to published at its true value.
func viewWithExtraPublished(prot *Protection, extra cellPos) *PublishedTable {
	pub := prot.Published()
	pub.Cells[extra.r][extra.c] = PublishedCell{Value: prot.Counts.Cells[extra.r][extra.c]}
	return pub
}
