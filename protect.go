package cellprotect

import (
	"fmt"
	"strings"
)

// Protect decides which cells to suppress in a count table.
//
// Primary suppressions are exactly the cells whose true value is in 1..4.
// Complementary suppressions are then added greedily (deterministic,
// row-major tie-break) until every primary cell is protected: its feasible
// range must allow 0 or a value >= 5. A final pass removes any complementary
// suppression that is not individually necessary. If suppressing every cell
// still fails to protect some primary cell, the table is rejected.
func Protect(c Counts) InternalReport {
	m := len(c.Cells)
	n := 0
	if m > 0 {
		n = len(c.Cells[0])
	}

	rep := InternalReport{Counts: c}
	rep.Suppression = make([][]SuppressionKind, m)
	for i := range rep.Suppression {
		rep.Suppression[i] = make([]SuppressionKind, n)
	}
	if m == 0 {
		return rep
	}

	mask := make([][]bool, m)
	primaryEdges := make([][2]int, 0)
	for i := 0; i < m; i++ {
		mask[i] = make([]bool, n)
		for j := 0; j < n; j++ {
			if v := c.Cells[i][j]; v >= 1 && v <= K-1 {
				mask[i][j] = true
				rep.Suppression[i][j] = Primary
				primaryEdges = append(primaryEdges, [2]int{i, j})
			}
		}
	}
	if len(primaryEdges) == 0 {
		return rep
	}

	// Reject if even suppressing every internal cell fails.
	full := make([][]bool, m)
	for i := range full {
		full[i] = make([]bool, n)
		for j := range full[i] {
			full[i][j] = true
		}
	}
	if bad := violatedEdges(buildNetwork(c, full), full, primaryEdges); len(bad) > 0 {
		rep.Rejected = true
		rep.RejectReason = rejectReason(c, bad)
		return rep
	}

	// Greedily add complementary suppressions.
	added := make([][2]int, 0)
	for {
		net := buildNetwork(c, mask)
		bad := violatedEdges(net, mask, primaryEdges)
		if len(bad) == 0 {
			break
		}
		best := -1
		bestFixed := -1
		for i := 0; i < m; i++ {
			for j := 0; j < n; j++ {
				if mask[i][j] {
					continue
				}
				mask[i][j] = true
				trial := buildNetwork(c, mask)
				fixed := len(bad) - len(violatedEdges(trial, mask, bad))
				mask[i][j] = false
				if fixed > bestFixed {
					bestFixed = fixed
					best = i*n + j
				}
			}
		}
		if best < 0 || bestFixed == 0 {
			// No single published cell fixes a primary: finish with the
			// (feasible) full mask; the minimality pass trims it.
			for i := 0; i < m; i++ {
				for j := 0; j < n; j++ {
					if !mask[i][j] {
						mask[i][j] = true
						added = append(added, [2]int{i, j})
					}
				}
			}
			break
		}
		i, j := best/n, best%n
		mask[i][j] = true
		added = append(added, [2]int{i, j})
	}

	// Minimality pass (reverse addition order): remove every complementary
	// suppression that is not individually necessary.
	for k := len(added) - 1; k >= 0; k-- {
		e := added[k]
		if rep.Suppression[e[0]][e[1]] == Primary {
			continue
		}
		mask[e[0]][e[1]] = false
		if bad := violatedEdges(buildNetwork(c, mask), mask, primaryEdges); len(bad) > 0 {
			mask[e[0]][e[1]] = true
		}
	}

	for i := 0; i < m; i++ {
		for j := 0; j < n; j++ {
			if !mask[i][j] {
				rep.Suppression[i][j] = Published
			} else if rep.Suppression[i][j] != Primary {
				rep.Suppression[i][j] = Complementary
			}
		}
	}
	return rep
}

// violatedEdges returns the edges (subset of want) whose feasible range is
// wholly inside 1..4 under the current mask.
func violatedEdges(net flowNetwork, mask [][]bool, want [][2]int) [][2]int {
	index := edgeIndex(mask)
	bad := make([][2]int, 0)
	for _, e := range want {
		idx := index[e[0]][e[1]]
		if idx < 0 {
			continue
		}
		lo, hi := net.edgeRange(idx)
		if lo >= 1 && hi <= K-1 {
			bad = append(bad, e)
		}
	}
	return bad
}

// edgeIndex maps suppressed cells to their index in the network edge list.
func edgeIndex(mask [][]bool) [][]int {
	m := len(mask)
	idx := make([][]int, m)
	cur := 0
	for i := range mask {
		idx[i] = make([]int, len(mask[i]))
		for j := range mask[i] {
			if mask[i][j] {
				idx[i][j] = cur
				cur++
			} else {
				idx[i][j] = -1
			}
		}
	}
	return idx
}

func rejectReason(c Counts, bad [][2]int) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("cannot publish: %d primary cell(s) remain identifiable in 1..4 even with every cell suppressed:", len(bad)))
	for _, e := range bad {
		b.WriteString(fmt.Sprintf(" %s/%s", c.Rows[e[0]], c.Cols[e[1]]))
	}
	return b.String()
}
