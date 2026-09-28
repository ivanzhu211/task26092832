package cellprotect

import "testing"

func rec(id, cip, ethnicity, status string) Record {
	return Record{StudentID: id, CIP: cip, Ethnicity: ethnicity, Gender: "Female", FirstGen: "No", Status: status}
}

func TestBuildReportCounts(t *testing.T) {
	records := []Record{
		rec("s1", "14.0901", "Asian", StatusApproved),
		rec("s1", "14.0901", "Asian", StatusDisbursed), // duplicate student
		rec("s2", "14.0901", "Asian", "Pending"),       // not an award
		rec("s3", "14.0201", "White", StatusDisbursed),
		rec("s4", "14.0201", "", StatusApproved), // undeclared ethnicity
		rec("s5", "52.0101", "White", StatusApproved),
		rec("s6", "52.0201", "Hispanic", StatusApproved),
		rec("s7", "52.0201", "Black", StatusApproved),
		rec("s8", "52.0201", "White", StatusApproved),
		rec("s9", "52.0201", "Asian", StatusApproved),
		rec("s10", "52.0201", "TwoOrMore", StatusApproved),
	}
	pub, err := BuildReport(records, Ethnicity)
	if err != nil {
		t.Fatalf("BuildReport: %v", err)
	}
	// Rows: only departments 14 and 52, sorted.
	if len(pub.Rows) != 2 || pub.Rows[0] != "14" || pub.Rows[1] != "52" {
		t.Fatalf("rows = %v, want [14 52]", pub.Rows)
	}
	// Columns: the full fixed ethnicity enumeration, including zero-count
	// values and Undeclared.
	if len(pub.Cols) != len(attributeValues[Ethnicity]) {
		t.Fatalf("cols = %v", pub.Cols)
	}
	if pub.Cols[len(pub.Cols)-1] != Undeclared {
		t.Fatalf("last column = %q, want %q", pub.Cols[len(pub.Cols)-1], Undeclared)
	}
	// Totals are always published: dept 14 has 2 awardees (s1, s3, s4 is
	// 3? s1, s3, s4 -> 3), dept 52 has 6.
	if pub.RowTotals[0] != 3 || pub.RowTotals[1] != 6 || pub.Total != 9 {
		t.Fatalf("totals = %v %v %v", pub.RowTotals, pub.ColTotals, pub.Total)
	}
	// Column totals must be consistent with the row totals.
	sum := 0
	for _, v := range pub.ColTotals {
		sum += v
	}
	if sum != pub.Total {
		t.Fatalf("column totals sum %d != total %d", sum, pub.Total)
	}
	// Every suppressed cell must hide its value.
	for r := range pub.Cells {
		for c := range pub.Cells[r] {
			if pub.Cells[r][c].Suppressed && pub.Cells[r][c].Value != 0 {
				t.Fatalf("suppressed cell (%d,%d) leaks value", r, c)
			}
		}
	}
}

func TestBuildReportRejectsUnprotectable(t *testing.T) {
	// A single department yields a one-row table in which every cell is
	// pinned by its column total; a small count forces rejection.
	records := []Record{
		rec("s1", "14.0901", "Asian", StatusApproved),
		rec("s2", "14.0901", "Asian", StatusApproved),
	}
	if _, err := BuildReport(records, Ethnicity); err == nil {
		t.Fatal("expected rejection for unprotectable one-row table")
	}
}

func TestBuildReportUnknownAttribute(t *testing.T) {
	if _, err := BuildReport(nil, Attribute(99)); err == nil {
		t.Fatal("expected error for unknown attribute")
	}
}
