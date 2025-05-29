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

func (p *ProgressLogger) InfoWithActor(actor string, msg string, args ...any) {
	p.base.InfoWithActor(actor, msg, args...)
}

func (p *ProgressLogger) WarnWithActor(actor string, msg string, args ...any) {
	p.base.WarnWithActor(actor, msg, args...)
}

func (p *ProgressLogger) ErrorWithActor(actor string, msg string, args ...any) {
	p.base.ErrorWithActor(actor, msg, args...)
}

func (p *ProgressLogger) Info(msg string, args ...any) {
	p.InfoWithActor("System", msg, args...)
}

func (p *ProgressLogger) Warn(msg string, args ...any) {
	p.WarnWithActor("System", msg, args...)
}

func (p *ProgressLogger) Error(msg string, args ...any) {
	p.ErrorWithActor("System", msg, args...)
}

func (p *ProgressLogger) ProgressRows() []iface.ProgressRow {
	return p.tracker.ProgressRows()
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
