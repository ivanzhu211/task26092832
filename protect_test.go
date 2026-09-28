package cellprotect

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// randomCounts builds a random count table with up to 4 rows, 4 columns
// and a grand total of at most 12.
func randomCounts(rng *rand.Rand) *Counts {
	nRows := 1 + rng.Intn(4)
	nCols := 1 + rng.Intn(4)
	counts := &Counts{Cells: make([][]int, nRows)}
	for r := 0; r < nRows; r++ {
		counts.Rows = append(counts.Rows, fmt.Sprintf("dept%d", r))
		counts.Cells[r] = make([]int, nCols)
	}
	for c := 0; c < nCols; c++ {
		counts.Cols = append(counts.Cols, fmt.Sprintf("val%d", c))
	}
	total := rng.Intn(13)
	for i := 0; i < total; i++ {
		counts.Cells[rng.Intn(nRows)][rng.Intn(nCols)]++
	}
	return counts
}

func isPrimaryValue(v int) bool { return v >= 1 && v < k }

// rangeViolates reports whether a feasible range violates the protection
// standard for a primary cell.
func rangeViolates(rng [2]int) bool { return rng[0] >= 1 && rng[1] <= k-1 }

func TestProtectRandomTables(t *testing.T) {
	rng := rand.New(rand.NewSource(20260928))
	for trial := 0; trial < 1000; trial++ {
		counts := randomCounts(rng)
		prot := Protect(counts)
		// Determinism: identical input must give identical output.
		prot2 := Protect(counts)
		if !reflect.DeepEqual(prot, prot2) {
			t.Fatalf("trial %d: Protect not deterministic", trial)
		}
		if prot.Rejected {
			verifyRejection(t, trial, counts)
			continue
		}
		pub := prot.Published()
		brute := bruteForceRanges(pub)
		for r := 0; r < len(counts.Rows); r++ {
			for c := 0; c < len(counts.Cols); c++ {
				cell := pub.Cells[r][c]
				state := prot.State[r][c]
				if state == StatePublished {
					if cell.Suppressed || cell.Value != counts.Cells[r][c] {
						t.Fatalf("trial %d: published cell (%d,%d) mangled", trial, r, c)
					}
					continue
				}
				if !cell.Suppressed || cell.Value != 0 {
					t.Fatalf("trial %d: suppressed cell (%d,%d) leaks value", trial, r, c)
				}
				// FeasibleRange must match brute force exactly.
				lo, hi := FeasibleRange(pub, r, c)
				want := brute[cellPos{r, c}]
				if lo != want[0] || hi != want[1] {
					t.Fatalf("trial %d: FeasibleRange(%d,%d) = [%d,%d], brute force [%d,%d]",
						trial, r, c, lo, hi, want[0], want[1])
				}
				// Every primary cell must satisfy the protection standard.
				if state == StatePrimary && rangeViolates(want) {
					t.Fatalf("trial %d: primary cell (%d,%d) range %v violates standard",
						trial, r, c, want)
				}
			}
		}
		// Minimality: publishing any single complementary cell must make
		// at least one primary cell violate the standard.
		for r := 0; r < len(counts.Rows); r++ {
			for c := 0; c < len(counts.Cols); c++ {
				if prot.State[r][c] != StateComplementary {
					continue
				}
				view := viewWithExtraPublished(prot, cellPos{r, c})
				ranges := bruteForceRanges(view)
				foundViolation := false
				for i := 0; i < len(counts.Rows); i++ {
					for j := 0; j < len(counts.Cols); j++ {
						if prot.State[i][j] != StatePrimary {
							continue
						}
						if rangeViolates(ranges[cellPos{i, j}]) {
							foundViolation = true
						}
					}
				}
				if !foundViolation {
					t.Fatalf("trial %d: complementary cell (%d,%d) is unnecessary", trial, r, c)
				}
			}
		}
	}
}

// verifyRejection checks that a rejected table genuinely cannot be
// protected: even with every cell suppressed, some primary cell's feasible
// range still lies entirely in [1, k-1].
func verifyRejection(t *testing.T, trial int, counts *Counts) {
	t.Helper()
	view := allSuppressedView(counts)
	ranges := bruteForceRanges(view)
	for r := 0; r < len(counts.Rows); r++ {
		for c := 0; c < len(counts.Cols); c++ {
			if isPrimaryValue(counts.Cells[r][c]) && rangeViolates(ranges[cellPos{r, c}]) {
				return
			}
		}
	}
	t.Fatalf("trial %d: table rejected but full suppression would satisfy the standard", trial)
}

func TestProtectPerformance(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	nRows, nCols := 40, 8
	counts := &Counts{Cells: make([][]int, nRows)}
	for r := 0; r < nRows; r++ {
		counts.Rows = append(counts.Rows, fmt.Sprintf("%02d", r))
		counts.Cells[r] = make([]int, nCols)
	}
	for c := 0; c < nCols; c++ {
		counts.Cols = append(counts.Cols, fmt.Sprintf("val%d", c))
	}
	// Aim for roughly 80 primary cells: most cells 0 or >= 5, about a
	// quarter in [1, 4].
	for r := 0; r < nRows; r++ {
		for c := 0; c < nCols; c++ {
			switch x := rng.Intn(100); {
			case x < 25:
				counts.Cells[r][c] = 1 + rng.Intn(4)
			case x < 40:
				counts.Cells[r][c] = 0
			default:
				counts.Cells[r][c] = 5 + rng.Intn(40)
			}
		}
	}
	start := time.Now()
	prot := Protect(counts)
	elapsed := time.Since(start)
	if prot.Rejected {
		t.Fatalf("unexpected rejection: %s", prot.Reason)
	}
	if elapsed > 30*time.Second {
		t.Fatalf("Protect took %v, limit is 30s", elapsed)
	}
	t.Logf("Protect on 40x8 table took %v", elapsed)
}
