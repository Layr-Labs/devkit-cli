package logger

import (
	"strings"

	"go.uber.org/zap"
)

type ZapLogger struct {
	log *zap.SugaredLogger
}

func NewZapLogger() *ZapLogger {
	logger, _ := zap.NewDevelopment()
	return &ZapLogger{log: logger.Sugar()}
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
