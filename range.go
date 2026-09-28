package cellprotect

// FeasibleRange computes the minimum and maximum possible values of the
// suppressed cell at (row, col), using only the published form: published
// cell values, all totals, and non-negativity of the suppressed cells.
func FeasibleRange(published *PublishedTable, row, col int) (min, max int) {
	nRows := len(published.Rows)
	nCols := len(published.Cols)
	if !published.Cells[row][col].Suppressed {
		v := published.Cells[row][col].Value
		return v, v
	}
	rowRes := make([]int, nRows)
	colRes := make([]int, nCols)
	suppressed := make([][]bool, nRows)
	for r := 0; r < nRows; r++ {
		rowRes[r] = published.RowTotals[r]
		suppressed[r] = make([]bool, nCols)
	}
	for c := 0; c < nCols; c++ {
		colRes[c] = published.ColTotals[c]
	}
	for r := 0; r < nRows; r++ {
		for c := 0; c < nCols; c++ {
			cell := published.Cells[r][c]
			if cell.Suppressed {
				suppressed[r][c] = true
			} else {
				rowRes[r] -= cell.Value
				colRes[c] -= cell.Value
			}
		}
	}
	return cellRange(nRows, nCols, rowRes, colRes, suppressed, row, col)
}

// cellRange computes tight bounds for the suppressed cell (row, col) given
// residual row/column margins (published cells already subtracted) and the
// set of suppressed cells. The transportation problem decomposes into
// connected components of the suppressed-cell bipartite graph, so only the
// component containing the target cell is solved, by min-cost flow.
func cellRange(nRows, nCols int, rowRes, colRes []int, suppressed [][]bool, row, col int) (min, max int) {
	compRows, compCols := component(suppressed, nRows, nCols, row, col)
	// Index component nodes: source, rows, cols, sink.
	rowNode := make([]int, nRows)
	colNode := make([]int, nCols)
	n := 1
	total := 0
	for r := 0; r < nRows; r++ {
		if compRows[r] {
			rowNode[r] = n
			n++
			total += rowRes[r]
		}
	}
	for c := 0; c < nCols; c++ {
		if compCols[c] {
			colNode[c] = n
			n++
		}
	}
	sink := n
	n++
	build := func(targetCost int) *flowGraph {
		fg := newFlowGraph(n)
		for r := 0; r < nRows; r++ {
			if compRows[r] {
				fg.addEdge(0, rowNode[r], rowRes[r], 0)
			}
		}
		for c := 0; c < nCols; c++ {
			if compCols[c] {
				fg.addEdge(colNode[c], sink, colRes[c], 0)
			}
		}
		for r := 0; r < nRows; r++ {
			if !compRows[r] {
				continue
			}
			for c := 0; c < nCols; c++ {
				if !suppressed[r][c] {
					continue
				}
				cost := 0
				if r == row && c == col {
					cost = targetCost
				}
				fg.addEdge(rowNode[r], colNode[c], total, cost)
			}
		}
		return fg
	}
	min = build(1).minCostFlow(0, sink, total)
	max = -build(-1).minCostFlow(0, sink, total)
	return min, max
}

// component returns the rows and columns of the connected component of the
// suppressed-cell bipartite graph that contains (row, col).
func component(suppressed [][]bool, nRows, nCols, row, col int) (compRows, compCols []bool) {
	compRows = make([]bool, nRows)
	compCols = make([]bool, nCols)
	compRows[row] = true
	compCols[col] = true
	for changed := true; changed; {
		changed = false
		for r := 0; r < nRows; r++ {
			for c := 0; c < nCols; c++ {
				if !suppressed[r][c] {
					continue
				}
				if compRows[r] && !compCols[c] {
					compCols[c] = true
					changed = true
				}
				if compCols[c] && !compRows[r] {
					compRows[r] = true
					changed = true
				}
			}
		}
	}
	return compRows, compCols
}
