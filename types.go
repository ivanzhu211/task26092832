// Package cellprotect implements small-cell suppression for demographic
// reports of scholarship awardees. It uses only the Go standard library.
package cellprotect

// Attribute selects which demographic dimension a report crosses with
// department.
type Attribute int

const (
	Ethnicity Attribute = iota
	Gender
	FirstGeneration
)

// Undeclared is the required "not reported" value present in every
// attribute's fixed value enumeration.
const Undeclared = "Undeclared"

// attributeValues lists the fixed column values per attribute, in report
// order. Zero-count values are still listed.
var attributeValues = map[Attribute][]string{
	Ethnicity:       {"Hispanic", "Asian", "Black", "White", "TwoOrMore", Undeclared},
	Gender:          {"Female", "Male", "Nonbinary", Undeclared},
	FirstGeneration: {"Yes", "No", Undeclared},
}

// Award record statuses that count as an award.
const (
	StatusApproved  = "Approved"
	StatusDisbursed = "Disbursed"
)

// Record is one input award record.
type Record struct {
	StudentID string
	CIP       string // e.g. "14.0901"; department is the first two characters
	Ethnicity string
	Gender    string
	FirstGen  string
	Status    string
}

// Counts is a cross-tabulation of awardee counts: rows are departments,
// columns are attribute values.
type Counts struct {
	Rows  []string
	Cols  []string
	Cells [][]int // [row][col]
}

// RowTotal returns the total count in row r.
func (c *Counts) RowTotal(r int) int {
	total := 0
	for _, v := range c.Cells[r] {
		total += v
	}
	return total
}

// ColTotal returns the total count in column col.
func (c *Counts) ColTotal(col int) int {
	total := 0
	for r := range c.Cells {
		total += c.Cells[r][col]
	}
	return total
}

// Total returns the grand total.
func (c *Counts) Total() int {
	total := 0
	for r := range c.Cells {
		total += c.RowTotal(r)
	}
	return total
}

// CellState is the internal per-cell protection decision.
type CellState int

const (
	StatePublished CellState = iota
	StatePrimary
	StateComplementary
)

// Protection is the internal result of Protect: every cell is marked
// published, primarily suppressed, or complementarily suppressed.
type Protection struct {
	Counts   *Counts
	State    [][]CellState
	Rejected bool
	Reason   string
}

// PublishedCell is the externally visible form of one cell: either a
// published value, or suppressed (with no reason given).
type PublishedCell struct {
	Suppressed bool
	Value      int
}

// PublishedTable is the externally visible report. Row, column and grand
// totals are always published.
type PublishedTable struct {
	Rows      []string
	Cols      []string
	Cells     [][]PublishedCell
	RowTotals []int
	ColTotals []int
	Total     int
}
