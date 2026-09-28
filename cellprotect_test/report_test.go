package cellprotect_test

import (
	"testing"

	cp "cellprotect"
)

func TestBuildReportBasics(t *testing.T) {
	rec := func(id, cip, eth, status string) cp.Record {
		return cp.Record{StudentID: id, CIPCode: cip, Ethnicity: eth, Status: status}
	}
	records := []cp.Record{
		rec("a1", "14.0901", "亚裔", cp.StatusApproved),
		rec("a2", "14.0901", "亚裔", cp.StatusApproved),
		rec("a3", "14.0901", "亚裔", cp.StatusApproved),
		rec("a1", "14.0901", "亚裔", cp.StatusDisbursed), // duplicate student, ignored
		rec("b1", "14.9999", "白人", cp.StatusDisbursed),
		rec("b2", "14.9999", "白人", cp.StatusDisbursed),
		rec("b3", "14.9999", "白人", cp.StatusDisbursed),
		rec("p1", "52.0201", "白人", "Pending"),          // excluded by status
		rec("u1", "52.0000", "???", cp.StatusApproved), // undeclared, other dept
		rec("u2", "52.0000", "???", cp.StatusApproved),
		rec("u3", "52.0000", "???", cp.StatusApproved),
		rec("u4", "52.0000", "???", cp.StatusApproved),
		rec("u5", "52.0000", "???", cp.StatusApproved),
		rec("u6", "52.0000", "???", cp.StatusApproved),
	}

	internal, pub := cp.BuildReport(records, cp.Ethnicity)
	if internal == nil || pub == nil {
		t.Fatal("expected published report")
	}
	if len(internal.Rows) != 2 || internal.Rows[0] != "14" || internal.Rows[1] != "52" {
		t.Fatalf("unexpected rows: %v", internal.Rows)
	}
	undeclared, asian, white := -1, -1, -1
	for j, v := range internal.Cols {
		switch v {
		case "未申报":
			undeclared = j
		case "亚裔":
			asian = j
		case "白人":
			white = j
		}
	}
	if undeclared < 0 || asian < 0 || white < 0 {
		t.Fatalf("enumeration missing values: %v", internal.Cols)
	}
	if internal.Cells[0][asian] != 3 || internal.Cells[0][white] != 3 || internal.Cells[0][undeclared] != 0 {
		t.Fatalf("dept 14 counts wrong: %v", internal.Cells[0])
	}
	if internal.RowTotals[0] != 6 {
		t.Fatalf("row total %d", internal.RowTotals[0])
	}
	if internal.Cells[1][undeclared] != 6 {
		t.Fatalf("dept 52 undeclared count %d", internal.Cells[1][undeclared])
	}
	if internal.Total != 12 {
		t.Fatalf("total %d", internal.Total)
	}

	// Totals always published.
	if pub.Total != internal.Total {
		t.Fatal("total mismatch")
	}

	// Zero-count columns still listed.
	if len(pub.Cells[0]) != len(internal.Cols) {
		t.Fatal("zero columns dropped")
	}
}

func TestBuildReportAttributes(t *testing.T) {
	mkRec := func(id, cip, gender, firstgen string) cp.Record {
		return cp.Record{StudentID: id, CIPCode: cip, Gender: gender, FirstGen: firstgen, Status: cp.StatusApproved}
	}
	// Two departments, five men each: no cell in 1..4, table publishes.
	records := []cp.Record{
		mkRec("a1", "14.0000", "男", "是"), mkRec("a2", "14.0000", "男", "是"),
		mkRec("a3", "14.0000", "男", "是"), mkRec("a4", "14.0000", "男", "是"),
		mkRec("a5", "14.0000", "男", "是"),
		mkRec("b1", "52.0000", "男", "否"), mkRec("b2", "52.0000", "男", "否"),
		mkRec("b3", "52.0000", "男", "否"), mkRec("b4", "52.0000", "男", "否"),
		mkRec("b5", "52.0000", "男", "否"),
	}
	_, pg := cp.BuildReport(records, cp.Gender)
	if len(pg.Cols) != 4 {
		t.Fatalf("gender cols: %v", pg.Cols)
	}
	_, pf := cp.BuildReport(records, cp.FirstGen)
	if len(pf.Cols) != 3 {
		t.Fatalf("firstgen cols: %v", pf.Cols)
	}
}

func TestRejectedReport(t *testing.T) {
	// One department, column total 3 with the residual concentrated so the
	// primary is pinned to 1..4 even under full suppression.
	c := cp.Counts{
		Rows:      []string{"D1"},
		Cols:      []string{"a", "b"},
		Cells:     [][]int{{1, 2}},
		RowTotals: []int{3},
		ColTotals: []int{1, 2},
		Total:     3,
	}
	internal := cp.Protect(c)
	if !internal.Rejected || internal.RejectReason == "" {
		t.Fatal("expected rejection with reason")
	}

	allSupp := allSuppressedReport(c)
	lo, hi, err := cp.FeasibleRange(allSupp, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !(lo >= 1 && hi <= 4) {
		t.Fatalf("rejected cell unexpectedly protected [%d,%d]", lo, hi)
	}
}

func TestDeterministic(t *testing.T) {
	rng := newRng(7)
	for it := 0; it < 50; it++ {
		c := randCounts(rng, 1+rng.Intn(6), 1+rng.Intn(6), rng.Intn(40))
		a := cp.Protect(c)
		b := cp.Protect(c)
		if a.Rejected != b.Rejected || a.RejectReason != b.RejectReason {
			t.Fatal("rejection differs")
		}
		for i := range a.Suppression {
			for j := range a.Suppression[i] {
				if a.Suppression[i][j] != b.Suppression[i][j] {
					t.Fatal("suppression differs")
				}
			}
		}
	}
}
