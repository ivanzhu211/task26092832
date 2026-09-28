package cellprotect

import (
	"fmt"
	"sort"
)

// BuildReport counts awardees per (department, attribute value) and then
// protects the table. It returns the publishable table, or an error if the
// table cannot be published even with all cells suppressed.
func BuildReport(records []Record, attribute Attribute) (*PublishedTable, error) {
	values, ok := attributeValues[attribute]
	if !ok {
		return nil, fmt.Errorf("unknown attribute: %d", attribute)
	}
	valueIndex := make(map[string]int, len(values))
	for i, v := range values {
		valueIndex[v] = i
	}
	// Keep awardees only, one record per student.
	type student struct {
		dept string
		attr string
	}
	seen := make(map[string]bool)
	var students []student
	for _, rec := range records {
		if rec.Status != StatusApproved && rec.Status != StatusDisbursed {
			continue
		}
		if seen[rec.StudentID] {
			continue
		}
		seen[rec.StudentID] = true
		if len(rec.CIP) < 2 {
			return nil, fmt.Errorf("student %s: invalid CIP code %q", rec.StudentID, rec.CIP)
		}
		attr := rec.attrValue(attribute)
		if _, ok := valueIndex[attr]; !ok {
			attr = Undeclared
		}
		students = append(students, student{dept: rec.CIP[:2], attr: attr})
	}
	// Departments with at least one awardee, sorted.
	deptSet := make(map[string]bool)
	for _, s := range students {
		deptSet[s.dept] = true
	}
	var depts []string
	for d := range deptSet {
		depts = append(depts, d)
	}
	sort.Strings(depts)
	deptIndex := make(map[string]int, len(depts))
	for i, d := range depts {
		deptIndex[d] = i
	}
	counts := &Counts{
		Rows:  depts,
		Cols:  values,
		Cells: make([][]int, len(depts)),
	}
	for i := range counts.Cells {
		counts.Cells[i] = make([]int, len(values))
	}
	for _, s := range students {
		counts.Cells[deptIndex[s.dept]][valueIndex[s.attr]]++
	}
	protection := Protect(counts)
	if protection.Rejected {
		return nil, fmt.Errorf("report cannot be published: %s", protection.Reason)
	}
	return protection.Published(), nil
}

// attrValue extracts the record's value for the given attribute.
func (r *Record) attrValue(attribute Attribute) string {
	switch attribute {
	case Ethnicity:
		return r.Ethnicity
	case Gender:
		return r.Gender
	case FirstGeneration:
		return r.FirstGen
	}
	return ""
}
