package logger

import (
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
)

type ProgressLogger struct {
	base    iface.Logger          // core Zap logger
	tracker iface.ProgressTracker // TTY or Log tracker
}

func NewProgressLogger(base iface.Logger, tracker iface.ProgressTracker) *ProgressLogger {
	return &ProgressLogger{
		base:    base,
		tracker: tracker,
	}
}

func (p *ProgressLogger) ProgressRows() []iface.ProgressRow {
	return p.tracker.ProgressRows()
}

func (p *ProgressLogger) Title(msg string, args ...any) {
	p.base.Title(msg, args...)
}

func (p *ProgressLogger) Info(msg string, args ...any) {
	p.base.Info(msg, args...)
}

func (p *ProgressLogger) Warn(msg string, args ...any) {
	p.base.Warn(msg, args...)
}

func (p *ProgressLogger) Error(msg string, args ...any) {
	p.base.Error(msg, args...)
}

func (p *ProgressLogger) SetProgress(name string, percent int, displayText string) {
	p.tracker.Set(name, percent, displayText)
}

func (p *ProgressLogger) PrintProgress() {
	p.tracker.Render()
}

func (p *ProgressLogger) ClearProgress() {
	p.tracker.Clear()
}
