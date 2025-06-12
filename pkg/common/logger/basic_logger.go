package logger

import (
	"fmt"
	"log"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
)

type BasicLogger struct {
	verbose bool
}

func NewLogger(verbose bool) *BasicLogger {
	return &BasicLogger{
		verbose: verbose,
	}
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

func (l *BasicLogger) Title(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	log.Printf("\n%s\n", formatted)
}

func (l *BasicLogger) Info(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	lines := strings.Split(strings.TrimSuffix(formatted, "\n"), "\n")
	for _, line := range lines {
		log.Printf("%s", line)
	}
}

func (l *BasicLogger) Warn(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	lines := strings.Split(strings.TrimSuffix(formatted, "\n"), "\n")
	for _, line := range lines {
		log.Printf("[Warning] %s", line)
	}
}

func (l *BasicLogger) Error(msg string, args ...any) {
	formatted := fmt.Sprintf(msg, args...)
	lines := strings.Split(strings.TrimSuffix(formatted, "\n"), "\n")
	for _, line := range lines {
		log.Printf("[Error] %s", line)
	}
}

func (l *BasicLogger) Debug(msg string, args ...any) {
	// skip debug when !verbose
	if !l.verbose {
		return
	}

	// format the message once
	formatted := fmt.Sprintf(msg, args...)

	// split into lines
	lines := strings.Split(strings.TrimSuffix(formatted, "\n"), "\n")

	// print the lines with log
	for _, line := range lines {
		log.Printf("Debug: %s", line)
	}
}

// Actor-based methods
func (l *BasicLogger) TitleWithActor(actor iface.Actor, msg string, args ...any) {
	l.Title(msg, args...)
}

func (l *BasicLogger) InfoWithActor(actor iface.Actor, msg string, args ...any) {
	l.Info(msg, args...)
}

func (l *BasicLogger) WarnWithActor(actor iface.Actor, msg string, args ...any) {
	l.Warn(msg, args...)
}

func (l *BasicLogger) ErrorWithActor(actor iface.Actor, msg string, args ...any) {
	l.Error(msg, args...)
}

func (l *BasicLogger) DebugWithActor(actor iface.Actor, msg string, args ...any) {
	l.Debug(msg, args...)
}
