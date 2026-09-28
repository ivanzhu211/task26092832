package cellprotect_test

import (
	"errors"
	"testing"

	cp "cellprotect"
)

func TestEmptyAndSafe(t *testing.T) {
	internal, pub := cp.BuildReport(nil, cp.Gender)
	if internal == nil || pub == nil {
		t.Fatal("empty input should still yield a report")
	}
	if len(internal.Rows) != 0 {
		t.Fatal("no winners expected")
	}

	c := cp.Counts{
		Rows:      []string{"D1", "D2"},
		Cols:      []string{"a", "b"},
		Cells:     [][]int{{5, 10}, {7, 0}},
		RowTotals: []int{15, 7},
		ColTotals: []int{12, 10},
		Total:     22,
	}
	internal2 := cp.Protect(c)
	if internal2.Rejected {
		t.Fatal("safe table rejected")
	}
	for i := range c.Cells {
		for j := range c.Cells[i] {
			if internal2.Suppression[i][j] != cp.Published {
				t.Fatalf("no suppression expected at (%d,%d)", i, j)
			}
		}
	}
	pub2 := toPublished(c, internal2)
	if _, _, err := cp.FeasibleRange(pub2, 0, 0); !errors.Is(err, cp.ErrNotSuppressed) {
		t.Fatalf("expected ErrNotSuppressed, got %v", err)
	}
	if _, _, err := cp.FeasibleRange(pub2, 5, 5); !errors.Is(err, cp.ErrNotSuppressed) {
		t.Fatalf("expected ErrNotSuppressed out of range, got %v", err)
	}
	if pub2.Total != 22 {
		t.Fatal("total must always be published")
	}
}

func TestAllZeroColumnRanges(t *testing.T) {
	c := cp.Counts{
		Rows:      []string{"D1"},
		Cols:      []string{"a", "b"},
		Cells:     [][]int{{0, 0}},
		RowTotals: []int{0},
		ColTotals: []int{0, 0},
		Total:     0,
	}
	internal := cp.Protect(c)
	if internal.Rejected {
		t.Fatal("all-zero table rejected")
	}
}
