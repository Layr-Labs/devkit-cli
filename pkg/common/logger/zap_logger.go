package logger

import (
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"go.uber.org/zap"
)

type ZapLogger struct {
	log *zap.SugaredLogger
}

func NewZapLogger(verbose bool) *ZapLogger {
	var logger *zap.Logger

	if verbose {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}

	return &ZapLogger{log: logger.Sugar()}
}

func (l *ZapLogger) InfoWithActor(actor string, msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	l.log.Infow(msg, append([]any{"actor", actor}, args...)...)
}

func (l *ZapLogger) WarnWithActor(actor string, msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	l.log.Warnw(msg, append([]any{"actor", actor}, args...)...)
}

func (l *ZapLogger) ErrorWithActor(actor string, msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	l.log.Errorw(msg, append([]any{"actor", actor}, args...)...)
}

func (l *ZapLogger) Info(msg string, args ...any) {
	l.InfoWithActor("System", msg, args...)
}

func (l *ZapLogger) Warn(msg string, args ...any) {
	l.WarnWithActor("System", msg, args...)
}

func (l *ZapLogger) Error(msg string, args ...any) {
	l.ErrorWithActor("System", msg, args...)
}

func (l *ZapLogger) Debug(msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	l.log.Debugf(msg, args...)
}

// Actor-based methods - basic implementations without color (delegating to regular methods)
func (l *ZapLogger) TitleWithActor(actor iface.Actor, msg string, args ...any) {
	l.Title(msg, args...)
}

func (l *ZapLogger) InfoWithActor(actor iface.Actor, msg string, args ...any) {
	l.Info(msg, args...)
}

func (l *ZapLogger) WarnWithActor(actor iface.Actor, msg string, args ...any) {
	l.Warn(msg, args...)
}

func (l *ZapLogger) ErrorWithActor(actor iface.Actor, msg string, args ...any) {
	l.Error(msg, args...)
}

func (l *ZapLogger) DebugWithActor(actor iface.Actor, msg string, args ...any) {
	l.Debug(msg, args...)
}
