package logger

import (
	"fmt"
	"strings"

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
	// format the message once
	formatted := fmt.Sprintf("\n"+msg+"\n", args...)

	// split into lines
	lines := strings.Split(formatted, "\n")

	// print the lines with log
	for _, line := range lines {
		l.log.Infof("%s", line)
	}
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
