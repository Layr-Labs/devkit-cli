package logger

import (
	"fmt"
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

func (l *ZapLogger) Title(msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	l.log.Infof("\n%s\n", fmt.Sprintf(msg, args...))
}

func (l *ZapLogger) Info(msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	l.log.Infof(msg, args...)
}

func (l *ZapLogger) Warn(msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	l.log.Warnf(msg, args...)
}

func (l *ZapLogger) Error(msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	l.log.Errorf(msg, args...)
}

func (l *ZapLogger) Debug(msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	l.log.Debugf(msg, args...)
}

// Actor-based methods
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
