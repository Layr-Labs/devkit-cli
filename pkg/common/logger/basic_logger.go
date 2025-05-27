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

func (l *BasicLogger) Info(msg string, args ...any) {
	// format the message once
	formatted := fmt.Sprintf(msg, args...)

	// split into lines
	lines := strings.Split(strings.TrimSuffix(formatted, "\n"), "\n")

	// print the lines with log
	for _, line := range lines {
		log.Printf("%s", line)
	}
}

func (l *BasicLogger) Warn(msg string, args ...any) {
	// format the message once
	formatted := fmt.Sprintf(msg, args...)

	// split into lines
	lines := strings.Split(strings.TrimSuffix(formatted, "\n"), "\n")

	// print the lines with log
	for _, line := range lines {
		log.Printf("Warning: %s", line)
	}
}

func (l *BasicLogger) Error(msg string, args ...any) {
	// format the message once
	formatted := fmt.Sprintf(msg, args...)

	// split into lines
	lines := strings.Split(strings.TrimSuffix(formatted, "\n"), "\n")

	// print the lines with log
	for _, line := range lines {
		log.Printf("Error: %s", line)
	}
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
}
