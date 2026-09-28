package cellprotect

import "sort"

// Fixed attribute enumerations. Every enumeration contains "未申报".
var (
	ethnicityValues = []string{
		"白人",
		"黑人或非裔美国人",
		"亚裔",
		"西班牙裔或拉丁裔",
		"原住民或阿拉斯加原住民",
		"太平洋岛民",
		"多种族",
		"未申报",
	}
	genderValues = []string{
		"男",
		"女",
		"非二元",
		"未申报",
	}
	firstGenValues = []string{
		"是",
		"否",
		"未申报",
	}
)

// attributeValues returns the fixed enumeration for an attribute.
func attributeValues(a Attribute) []string {
	switch a {
	case Ethnicity:
		return ethnicityValues
	case Gender:
		return genderValues
	case FirstGen:
		return firstGenValues
	default:
		// Unknown attribute: only undeclared is possible.
		return []string{"未申报"}
	}
}

// normalize maps a raw attribute value into its fixed enumeration; anything
// unrecognized (including empty) maps to "未申报".
func normalize(a Attribute, raw string) string {
	for _, v := range attributeValues(a) {
		if v == raw {
			return raw
		}
	}
	return "未申报"
}

// department extracts the department code: the first two characters of CIP.
func department(cip string) string {
	if len(cip) >= 2 {
		return cip[:2]
	}
	return cip
}

// BuildReport counts unique award recipients per (department, attribute),
// protects the table, and returns the internal and published forms.
func BuildReport(records []Record, attribute Attribute) (*InternalReport, *PublishedReport) {
	cols := attributeValues(attribute)

	seen := make(map[string]struct{})
	type cellKey struct {
		dept string
		val  string
	}
	counts := make(map[cellKey]int)
	deptSet := make(map[string]struct{})

	for _, r := range records {
		if r.Status != StatusApproved && r.Status != StatusDisbursed {
			continue
		}
		if _, ok := seen[r.StudentID]; ok {
			continue
		}
		seen[r.StudentID] = struct{}{}

		dept := department(r.CIPCode)
		var val string
		switch attribute {
		case Ethnicity:
			val = normalize(attribute, r.Ethnicity)
		case Gender:
			val = normalize(attribute, r.Gender)
		case FirstGen:
			val = normalize(attribute, r.FirstGen)
		default:
			val = "未申报"
		}
		counts[cellKey{dept, val}]++
		deptSet[dept] = struct{}{}
	}

	depts := make([]string, 0, len(deptSet))
	for d := range deptSet {
		depts = append(depts, d)
	}
	sort.Strings(depts)

	m, n := len(depts), len(cols)
	c := Counts{
		Rows:      depts,
		Cols:      cols,
		Cells:     make([][]int, m),
		RowTotals: make([]int, m),
		ColTotals: make([]int, n),
	}
	for i, d := range depts {
		c.Cells[i] = make([]int, n)
		for j, v := range cols {
			vv := counts[cellKey{d, v}]
			c.Cells[i][j] = vv
			c.RowTotals[i] += vv
			c.ColTotals[j] += vv
			c.Total += vv
		}
	}

	internal := Protect(c)
	return &internal, publish(&internal)
}

// publish converts an internal report into its releasable form. Totals are
// always published; suppressed cells reveal no reason.
func publish(r *InternalReport) *PublishedReport {
	if r.Rejected {
		return nil
	}
	m, n := len(r.Cells), 0
	if m > 0 {
		n = len(r.Cells[0])
	}
	p := &PublishedReport{
		Rows:      append([]string(nil), r.Rows...),
		Cols:      append([]string(nil), r.Cols...),
		Cells:     make([][]PublishedCell, m),
		RowTotals: append([]int(nil), r.RowTotals...),
		ColTotals: append([]int(nil), r.ColTotals...),
		Total:     r.Total,
	}
	for i := range r.Cells {
		p.Cells[i] = make([]PublishedCell, n)
		for j := range r.Cells[i] {
			cell := PublishedCell{Value: r.Cells[i][j]}
			if r.Suppression[i][j] == Published {
				cell.Kind = CellPublished
			} else {
				cell.Kind = CellSuppressed
			}
			p.Cells[i][j] = cell
		}
	}
	return p
}
