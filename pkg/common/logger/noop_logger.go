package logger

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
)

// LogEntry represents a single log entry with level and message
type LogEntry struct {
	Level   string
	Message string
}

// NoopLogger implements the Logger interface but doesn't output anything.
// Instead, it buffers all log messages for testing assertions.
// It is safe for concurrent use.
type NoopLogger struct {
	mu      sync.RWMutex
	entries []LogEntry
}

// NewNoopLogger creates a new no-op logger for testing
func NewNoopLogger() *NoopLogger {
	return &NoopLogger{
		entries: make([]LogEntry, 0),
	}
}

// Title implements the Logger interface - buffers title messages
func (l *NoopLogger) Title(msg string, args ...any) {
	formatted := fmt.Sprintf("\n"+msg+"\n", args...)
	l.addEntry("TITLE", formatted)
}

// Info implements the Logger interface - buffers info messages
func (l *NoopLogger) Info(msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	l.addEntry("INFO", formatted)
}

// Warn implements the Logger interface - buffers warning messages
func (l *NoopLogger) Warn(msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	l.addEntry("WARN", formatted)
}

// Error implements the Logger interface - buffers error messages
func (l *NoopLogger) Error(msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	l.addEntry("ERROR", formatted)
}

// Debug implements the Logger interface - buffers debug messages
func (l *NoopLogger) Debug(msg string, args ...any) {
	msg = strings.Trim(msg, "\n")
	if msg == "" {
		return
	}
	formatted := fmt.Sprintf(msg, args...)
	l.addEntry("DEBUG", formatted)
}

// addEntry safely adds a log entry to the buffer
func (l *NoopLogger) addEntry(level, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, LogEntry{
		Level:   level,
		Message: message,
	})
}

// GetEntries returns a copy of all buffered log entries
func (l *NoopLogger) GetEntries() []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	entries := make([]LogEntry, len(l.entries))
	copy(entries, l.entries)
	return entries
}

// GetEntriesByLevel returns all entries for a specific log level
func (l *NoopLogger) GetEntriesByLevel(level string) []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var filtered []LogEntry
	for _, entry := range l.entries {
		if entry.Level == level {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// GetMessages returns all buffered messages as strings
func (l *NoopLogger) GetMessages() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	messages := make([]string, len(l.entries))
	for i, entry := range l.entries {
		messages[i] = entry.Message
	}
	return messages
}

// GetMessagesByLevel returns all messages for a specific log level
func (l *NoopLogger) GetMessagesByLevel(level string) []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var messages []string
	for _, entry := range l.entries {
		if entry.Level == level {
			messages = append(messages, entry.Message)
		}
	}
	return messages
}

// Clear removes all buffered entries
func (l *NoopLogger) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = l.entries[:0]
}

// Len returns the number of buffered entries
func (l *NoopLogger) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries)
}

// Contains checks if any log entry contains the specified text
func (l *NoopLogger) Contains(text string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, entry := range l.entries {
		if strings.Contains(entry.Message, text) {
			return true
		}
	}
	return false
}

// ContainsLevel checks if any log entry with the specified level contains the text
func (l *NoopLogger) ContainsLevel(level, text string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, entry := range l.entries {
		if entry.Level == level && strings.Contains(entry.Message, text) {
			return true
		}
	}
	return false
}

// NoopProgressTracker is a progress tracker that does nothing (for testing)
type NoopProgressTracker struct{}

// NewNoopProgressTracker creates a new no-op progress tracker
func NewNoopProgressTracker() *NoopProgressTracker {
	return &NoopProgressTracker{}
}

// ProgressRows returns an empty slice
func (n *NoopProgressTracker) ProgressRows() []iface.ProgressRow {
	return []iface.ProgressRow{}
}

// Set does nothing
func (n *NoopProgressTracker) Set(id string, pct int, label string) {
	// No-op
}

// Render does nothing
func (n *NoopProgressTracker) Render() {
	// No-op
}

// Clear does nothing
func (n *NoopProgressTracker) Clear() {
	// No-op
}
