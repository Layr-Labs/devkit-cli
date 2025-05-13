package iface

type ProgressTracker interface {
	Set(id string, pct int, label string)
	Render()
	Clear()
}

type ProgressInfo struct {
	Percentage  int
	DisplayText string
	Timestamp   string
}
