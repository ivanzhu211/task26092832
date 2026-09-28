// Package cellprotect implements small-cell protection for scholarship
// demographic reports.
package cellprotect

// Attribute selects which demographic attribute a report cross-tabulates.
type Attribute string

const (
	Ethnicity Attribute = "ethnicity"
	Gender    Attribute = "gender"
	FirstGen  Attribute = "firstgen"
)

// AwardStatus values.
const (
	StatusApproved  = "Approved"
	StatusDisbursed = "Disbursed"
)

// Record is one award record for a student.
type Record struct {
	StudentID string
	CIPCode   string // e.g. "14.0901"; department is its first two chars
	Ethnicity string
	Gender    string
	FirstGen  string
	Status    string
}

// SuppressionKind records why a cell is suppressed in the internal result.
type SuppressionKind int

const (
	Published SuppressionKind = iota
	Primary
	Complementary
)

// CellKind is the only cell state visible in a published report.
type CellKind int

const (
	CellPublished CellKind = iota
	CellSuppressed
)

// PublishedCell is one cell of a releasable table.
type PublishedCell struct {
	Value int
	Kind  CellKind
}

// InternalCell additionally records the suppression reason.
type InternalCell struct {
	Value int
	Kind  SuppressionKind
}

// Counts is an internal count table of m rows by n columns plus totals.
type Counts struct {
	Rows      []string // department codes
	Cols      []string // attribute values
	Cells     [][]int  // [row][col] counts
	RowTotals []int
	ColTotals []int
	Total     int
}

// InternalReport is the full result of Protect.
type InternalReport struct {
	Counts
	Suppression  [][]SuppressionKind // [row][col]
	Rejected     bool
	RejectReason string
}

// PublishedReport is the releasable shape.
type PublishedReport struct {
	Rows      []string
	Cols      []string
	Cells     [][]PublishedCell
	RowTotals []int
	ColTotals []int
	Total     int
}

// K is the protection threshold.
const K = 5
