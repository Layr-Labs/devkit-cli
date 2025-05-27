package iface

// ProgressRow is a snapshot of a completed progress update.
type ProgressRow struct {
	Module string
	Pct    int
	Label  string
}

type ProgressTracker interface {
	ProgressRows() []ProgressRow
	Set(id string, pct int, label string)
	Render()
	Clear()
}

type ProgressInfo struct {
	Percentage  int
	DisplayText string
	Timestamp   string
}
