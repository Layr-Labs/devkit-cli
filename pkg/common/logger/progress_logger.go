package logger

import (
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
)

type ProgressLogger struct {
	base    iface.Logger          // core Zap logger
	tracker iface.ProgressTracker // TTY or Log tracker
}

func NewProgressLogger(baseLogger iface.Logger, tracker iface.ProgressTracker) *ProgressLogger {
	return &ProgressLogger{
		base:    baseLogger,
		tracker: tracker,
	}
}

func (p *ProgressLogger) Title(msg string, args ...any) {
	p.tracker.Clear()
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

func (p *ProgressLogger) Debug(msg string, args ...any) {
	p.base.Debug(msg, args...)
}

// Actor-based methods
func (p *ProgressLogger) TitleWithActor(actor iface.Actor, msg string, args ...any) {
	p.tracker.Clear()
	p.base.TitleWithActor(actor, msg, args...)
}

func (p *ProgressLogger) InfoWithActor(actor iface.Actor, msg string, args ...any) {
	p.base.InfoWithActor(actor, msg, args...)
}

func (p *ProgressLogger) WarnWithActor(actor iface.Actor, msg string, args ...any) {
	p.base.WarnWithActor(actor, msg, args...)
}

func (p *ProgressLogger) ErrorWithActor(actor iface.Actor, msg string, args ...any) {
	p.base.ErrorWithActor(actor, msg, args...)
}

func (p *ProgressLogger) DebugWithActor(actor iface.Actor, msg string, args ...any) {
	p.base.DebugWithActor(actor, msg, args...)
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
