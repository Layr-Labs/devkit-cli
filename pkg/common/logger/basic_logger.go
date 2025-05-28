package logger

import (
	"fmt"
	"log"
	"strings"
)

type BasicLogger struct {
}

func NewLogger() *BasicLogger {
	return &BasicLogger{}
}

func colorForActor(actor string) string {
	switch actor {
	case "User":
		return "\033[32m" // Green
	case "AVS Developer":
		return "\033[34m" // Blue
	case "Operator":
		return "\033[33m" // Yellow
	case "System":
		return "\033[37m" // Gray
	default:
		return "\033[0m" // Reset
	}
}

func resetColor() string {
	return "\033[0m"
}

func (l *BasicLogger) InfoWithActor(actor string, msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	lines := strings.Split(strings.TrimSuffix(formatted, "\n"), "\n")
	color := colorForActor(actor)
	reset := resetColor()
	for _, line := range lines {
		log.Printf("%s[%s] %s%s", color, actor, line, reset)
	}
}

func (l *BasicLogger) WarnWithActor(actor string, msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	lines := strings.Split(strings.TrimSuffix(formatted, "\n"), "\n")
	color := colorForActor(actor)
	reset := resetColor()
	for _, line := range lines {
		log.Printf("%s[%s][Warning] %s%s", color, actor, line, reset)
	}
}

func (l *BasicLogger) ErrorWithActor(actor string, msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	lines := strings.Split(strings.TrimSuffix(formatted, "\n"), "\n")
	color := colorForActor(actor)
	reset := resetColor()
	for _, line := range lines {
		log.Printf("%s[%s][Error] %s%s", color, actor, line, reset)
	}
}

func (l *BasicLogger) Info(msg string, args ...any) {
	l.InfoWithActor("System", msg, args...)
}

func (l *BasicLogger) Warn(msg string, args ...any) {
	l.WarnWithActor("System", msg, args...)
}

func (l *BasicLogger) Error(msg string, args ...any) {
	l.ErrorWithActor("System", msg, args...)
<<<<<<< Updated upstream
}

func (l *BasicLogger) Debug(msg string, args ...any) {
	// format the message once
	formatted := fmt.Sprintf(msg, args...)

	// split into lines
	lines := strings.Split(strings.TrimSuffix(formatted, "\n"), "\n")

	// print the lines with log
	for _, line := range lines {
		log.Printf("Error: %s", line)
	}
=======
>>>>>>> Stashed changes
}
