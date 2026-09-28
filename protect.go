package cellprotect

import "fmt"

// k is the fixed disclosure threshold: counts in [1, k-1] are primary
// suppressions.
const k = 5

// Protect decides which cells of the count table to suppress. The result
// marks every cell published, primarily suppressed, or complementarily
// suppressed. If the protection standard cannot be met even with every
// cell suppressed, the table is rejected and Reason explains why.
func Protect(counts *Counts) *Protection {
	p := newProtector(counts)
	for r := 0; r < p.nRows; r++ {
		for c := 0; c < p.nCols; c++ {
			if v := counts.Cells[r][c]; v >= 1 && v < k {
				p.suppress(r, c, StatePrimary)
			}
		}
	}
	for {
		row, col, ok := p.firstViolation()
		if !ok {
			break
		}
		if !p.addComplementary(row, col) {
			return p.reject(row, col)
		}
	}
	p.prune()
	return p.result()
}

// Published converts the internal protection result into the externally
// publishable form, where each cell is either a published value or
// suppressed with no reason attached.
func (p *Protection) Published() *PublishedTable {
	if p == nil || p.Rejected {
		return nil
	}
	nRows := len(p.Counts.Rows)
	nCols := len(p.Counts.Cols)
	pub := &PublishedTable{
		Rows:      p.Counts.Rows,
		Cols:      p.Counts.Cols,
		Cells:     make([][]PublishedCell, nRows),
		RowTotals: make([]int, nRows),
		ColTotals: make([]int, nCols),
	}
	for r := 0; r < nRows; r++ {
		pub.Cells[r] = make([]PublishedCell, nCols)
		pub.RowTotals[r] = p.Counts.RowTotal(r)
		for c := 0; c < nCols; c++ {
			if p.State[r][c] == StatePublished {
				pub.Cells[r][c] = PublishedCell{Value: p.Counts.Cells[r][c]}
			} else {
				pub.Cells[r][c] = PublishedCell{Suppressed: true}
			}
		}
	}
	for c := 0; c < nCols; c++ {
		pub.ColTotals[c] = p.Counts.ColTotal(c)
	}
	pub.Total = p.Counts.Total()
	return pub
}

// protector holds the mutable suppression state during Protect.
type protector struct {
	counts *Counts
	nRows  int
	nCols  int
	state  [][]CellState
	sup    [][]bool
	rowRes []int // residual margins: totals minus published cells
	colRes []int
	cache  map[[2]int][2]int // cached feasible ranges of primary cells
}

func newProtector(counts *Counts) *protector {
	nRows := len(counts.Rows)
	nCols := len(counts.Cols)
	p := &protector{
		counts: counts,
		nRows:  nRows,
		nCols:  nCols,
		state:  make([][]CellState, nRows),
		sup:    make([][]bool, nRows),
		rowRes: make([]int, nRows),
		colRes: make([]int, nCols),
		cache:  make(map[[2]int][2]int),
	}
	for r := 0; r < nRows; r++ {
		p.state[r] = make([]CellState, nCols)
		p.sup[r] = make([]bool, nCols)
	}
	return p
}

// suppress marks a cell suppressed (primary or complementary) and updates
// residuals and the range cache.
func (p *protector) suppress(r, c int, st CellState) {
	p.state[r][c] = st
	p.sup[r][c] = true
	v := p.counts.Cells[r][c]
	p.rowRes[r] += v
	p.colRes[c] += v
	p.invalidateComponent(r, c)
}

// publish returns a suppressed cell to the published state.
func (p *protector) publish(r, c int) {
	p.invalidateComponent(r, c) // before removal, while the component is whole
	p.state[r][c] = StatePublished
	p.sup[r][c] = false
	v := p.counts.Cells[r][c]
	p.rowRes[r] -= v
	p.colRes[c] -= v
}

// invalidateComponent drops cached ranges of primary cells in the component
// containing (r, c); only those ranges can change when (r, c) changes state.
func (p *protector) invalidateComponent(r, c int) {
	compRows, compCols := component(p.sup, p.nRows, p.nCols, r, c)
	for cell := range p.cache {
		if compRows[cell[0]] && compCols[cell[1]] {
			delete(p.cache, cell)
		}
	}
}

// rangeOf returns the cached feasible range of a primary cell.
func (p *protector) rangeOf(r, c int) [2]int {
	if rng, ok := p.cache[[2]int{r, c}]; ok {
		return rng
	}
	lo, hi := cellRange(p.nRows, p.nCols, p.rowRes, p.colRes, p.sup, r, c)
	rng := [2]int{lo, hi}
	p.cache[[2]int{r, c}] = rng
	return rng
}

// violates reports whether a primary cell with the given feasible range
// fails the protection standard: its possible values must not all lie in
// [1, k-1]; it passes when the minimum can be 0 or the maximum can be >= k.
func violates(rng [2]int) bool {
	return rng[0] >= 1 && rng[1] <= k-1
}

// firstViolation returns the first (lowest-index) primary cell whose
// feasible range violates the protection standard.
func (p *protector) firstViolation() (row, col int, ok bool) {
	for r := 0; r < p.nRows; r++ {
		for c := 0; c < p.nCols; c++ {
			if p.state[r][c] == StatePrimary && violates(p.rangeOf(r, c)) {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

// addComplementary suppresses one more published cell to widen the feasible
// range of the violating primary cell (vr, vc). It returns false when no
// published cell can ever affect that range, meaning it is already final.
func (p *protector) addComplementary(vr, vc int) bool {
	compRows, compCols := component(p.sup, p.nRows, p.nCols, vr, vc)
	// A published cell can only join (and thus affect) the component of the
	// violating cell if it lies in a component row or column.
	var candidates [][2]int
	for r := 0; r < p.nRows; r++ {
		for c := 0; c < p.nCols; c++ {
			if p.state[r][c] == StatePublished && (compRows[r] || compCols[c]) {
				candidates = append(candidates, [2]int{r, c})
			}
		}
	}
	if len(candidates) == 0 {
		return false
	}
	cur := p.rangeOf(vr, vc)
	bestGain := 0
	best := [2]int{-1, -1}
	for _, q := range candidates {
		p.suppress(q[0], q[1], StateComplementary)
		next := p.rangeOf(vr, vc)
		p.publish(q[0], q[1])
		gain := (cur[0] - next[0]) + (next[1] - cur[1])
		if gain > bestGain {
			bestGain = gain
			best = q
		}
	}
	if bestGain == 0 {
		// No single suppression widens the range; suppress the first
		// candidate anyway so combined suppressions can take effect.
		best = candidates[0]
	}
	p.suppress(best[0], best[1], StateComplementary)
	return true
}

// prune removes unnecessary complementary suppressions until every
// remaining one is individually necessary: publishing any single one would
// make some primary cell violate the protection standard.
func (p *protector) prune() {
	for changed := true; changed; {
		changed = false
		for r := 0; r < p.nRows; r++ {
			for c := 0; c < p.nCols; c++ {
				if p.state[r][c] != StateComplementary {
					continue
				}
				affected := p.primariesInComponent(r, c)
				p.publish(r, c)
				if p.allPass(affected) {
					changed = true
				} else {
					p.suppress(r, c, StateComplementary)
				}
			}
		}
	}
}

// primariesInComponent lists the primary cells in the component of (r, c).
func (p *protector) primariesInComponent(r, c int) [][2]int {
	compRows, compCols := component(p.sup, p.nRows, p.nCols, r, c)
	var out [][2]int
	for i := 0; i < p.nRows; i++ {
		for j := 0; j < p.nCols; j++ {
			if p.state[i][j] == StatePrimary && compRows[i] && compCols[j] {
				out = append(out, [2]int{i, j})
			}
		}
	}
	return out
}

// allPass reports whether every listed primary cell satisfies the standard.
func (p *protector) allPass(primaries [][2]int) bool {
	for _, cell := range primaries {
		if violates(p.rangeOf(cell[0], cell[1])) {
			return false
		}
	}
	return true
}

func (p *protector) reject(row, col int) *Protection {
	return &Protection{
		Counts:   p.counts,
		State:    p.state,
		Rejected: true,
		Reason: fmt.Sprintf(
			"cell (%d, %d) with count %d cannot be protected: its feasible range stays within [1, %d] even with all remaining cells suppressed",
			row, col, p.counts.Cells[row][col], k-1),
	}
}

func (p *protector) result() *Protection {
	return &Protection{Counts: p.counts, State: p.state}
}
