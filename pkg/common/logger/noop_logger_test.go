package logger

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoopLogger_Interface(t *testing.T) {
	// Verify that NoopLogger implements the Logger interface
	var _ interface {
		Title(msg string, args ...any)
		Info(msg string, args ...any)
		Warn(msg string, args ...any)
		Error(msg string, args ...any)
		Debug(msg string, args ...any)
	} = &NoopLogger{}
}

func TestNoopLogger_NewNoopLogger(t *testing.T) {
	logger := NewNoopLogger()

	assert.NotNil(t, logger)
	assert.Equal(t, 0, logger.Len())
	assert.Empty(t, logger.GetEntries())
}

func TestNoopLogger_LoggingMethods(t *testing.T) {
	logger := NewNoopLogger()

	// Test each logging method
	logger.Title("Test Title %s", "message")
	logger.Info("Test Info %s", "message")
	logger.Warn("Test Warn %s", "message")
	logger.Error("Test Error %s", "message")
	logger.Debug("Test Debug %s", "message")

	// Verify entries were captured
	assert.Equal(t, 5, logger.Len())

	entries := logger.GetEntries()
	require.Len(t, entries, 5)

	// Check each entry
	assert.Equal(t, "TITLE", entries[0].Level)
	assert.Contains(t, entries[0].Message, "Test Title message")

	assert.Equal(t, "INFO", entries[1].Level)
	assert.Equal(t, "Test Info message", entries[1].Message)

	assert.Equal(t, "WARN", entries[2].Level)
	assert.Equal(t, "Test Warn message", entries[2].Message)

	assert.Equal(t, "ERROR", entries[3].Level)
	assert.Equal(t, "Test Error message", entries[3].Message)

	assert.Equal(t, "DEBUG", entries[4].Level)
	assert.Equal(t, "Test Debug message", entries[4].Message)
}

func TestNoopLogger_EmptyMessages(t *testing.T) {
	logger := NewNoopLogger()

	// Test empty messages are ignored for Info, Warn, Error, Debug
	logger.Info("")
	logger.Info("\n")
	logger.Info("\n\n")
	logger.Warn("")
	logger.Error("")
	logger.Debug("")

	// Title should still be captured even if empty
	logger.Title("")

	assert.Equal(t, 1, logger.Len())
	entries := logger.GetEntries()
	assert.Equal(t, "TITLE", entries[0].Level)
}

func TestNoopLogger_GetEntriesByLevel(t *testing.T) {
	logger := NewNoopLogger()

	logger.Info("info message 1")
	logger.Error("error message 1")
	logger.Info("info message 2")
	logger.Warn("warn message 1")
	logger.Error("error message 2")

	// Test filtering by level
	infoEntries := logger.GetEntriesByLevel("INFO")
	assert.Len(t, infoEntries, 2)
	assert.Equal(t, "info message 1", infoEntries[0].Message)
	assert.Equal(t, "info message 2", infoEntries[1].Message)

	errorEntries := logger.GetEntriesByLevel("ERROR")
	assert.Len(t, errorEntries, 2)
	assert.Equal(t, "error message 1", errorEntries[0].Message)
	assert.Equal(t, "error message 2", errorEntries[1].Message)

	warnEntries := logger.GetEntriesByLevel("WARN")
	assert.Len(t, warnEntries, 1)
	assert.Equal(t, "warn message 1", warnEntries[0].Message)

	debugEntries := logger.GetEntriesByLevel("DEBUG")
	assert.Len(t, debugEntries, 0)
}

func TestNoopLogger_GetMessages(t *testing.T) {
	logger := NewNoopLogger()

	logger.Info("first message")
	logger.Warn("second message")
	logger.Error("third message")

	messages := logger.GetMessages()
	expected := []string{"first message", "second message", "third message"}
	assert.Equal(t, expected, messages)
}

func TestNoopLogger_GetMessagesByLevel(t *testing.T) {
	logger := NewNoopLogger()

	logger.Info("info 1")
	logger.Error("error 1")
	logger.Info("info 2")
	logger.Error("error 2")

	infoMessages := logger.GetMessagesByLevel("INFO")
	assert.Equal(t, []string{"info 1", "info 2"}, infoMessages)

	errorMessages := logger.GetMessagesByLevel("ERROR")
	assert.Equal(t, []string{"error 1", "error 2"}, errorMessages)

	debugMessages := logger.GetMessagesByLevel("DEBUG")
	assert.Empty(t, debugMessages)
}

func TestNoopLogger_Clear(t *testing.T) {
	logger := NewNoopLogger()

	logger.Info("test message")
	logger.Error("another message")

	assert.Equal(t, 2, logger.Len())

	logger.Clear()

	assert.Equal(t, 0, logger.Len())
	assert.Empty(t, logger.GetEntries())
	assert.Empty(t, logger.GetMessages())
}

func TestNoopLogger_Contains(t *testing.T) {
	logger := NewNoopLogger()

	logger.Info("this is a test message")
	logger.Error("something went wrong")
	logger.Warn("be careful about this")

	// Test general contains
	assert.True(t, logger.Contains("test message"))
	assert.True(t, logger.Contains("went wrong"))
	assert.True(t, logger.Contains("careful"))
	assert.False(t, logger.Contains("not found"))

	// Test partial matches
	assert.True(t, logger.Contains("this is"))
	assert.True(t, logger.Contains("wrong"))
}

func TestNoopLogger_ContainsLevel(t *testing.T) {
	logger := NewNoopLogger()

	logger.Info("info message here")
	logger.Error("error message here")
	logger.Warn("warning message here")

	// Test level-specific contains
	assert.True(t, logger.ContainsLevel("INFO", "info message"))
	assert.True(t, logger.ContainsLevel("ERROR", "error message"))
	assert.True(t, logger.ContainsLevel("WARN", "warning message"))

	// Test cross-level (should return false)
	assert.False(t, logger.ContainsLevel("INFO", "error message"))
	assert.False(t, logger.ContainsLevel("ERROR", "info message"))
	assert.False(t, logger.ContainsLevel("DEBUG", "info message"))

	// Test non-existent text
	assert.False(t, logger.ContainsLevel("INFO", "not found"))
}

func TestNoopLogger_ConcurrentSafety(t *testing.T) {
	logger := NewNoopLogger()
	const numGoroutines = 100
	const messagesPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines writing to the logger
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				logger.Info("goroutine %d message %d", id, j)
				logger.Error("goroutine %d error %d", id, j)
			}
		}(i)
	}

	// Start a goroutine reading from the logger
	readDone := make(chan bool)
	go func() {
		for i := 0; i < 50; i++ {
			_ = logger.GetEntries()
			_ = logger.GetMessages()
			_ = logger.Len()
			_ = logger.Contains("goroutine")
		}
		readDone <- true
	}()

	wg.Wait()
	<-readDone

	// Verify all messages were captured
	expectedCount := numGoroutines * messagesPerGoroutine * 2 // 2 message types per iteration
	assert.Equal(t, expectedCount, logger.Len())

	// Test clearing while reading
	var clearWg sync.WaitGroup
	clearWg.Add(2)

	go func() {
		defer clearWg.Done()
		logger.Clear()
	}()

	go func() {
		defer clearWg.Done()
		_ = logger.GetEntries()
	}()

	clearWg.Wait()

	// After clear, should have 0 entries
	assert.Equal(t, 0, logger.Len())
}

func TestNoopLogger_MessageFormatting(t *testing.T) {
	logger := NewNoopLogger()

	// Test format strings with various argument types
	logger.Info("number: %d, string: %s, float: %.2f", 42, "test", 3.14159)
	logger.Error("user %s has %d items", "alice", 5)

	messages := logger.GetMessages()
	assert.Contains(t, messages[0], "number: 42, string: test, float: 3.14")
	assert.Contains(t, messages[1], "user alice has 5 items")
}

// Example test showing typical usage in unit tests
func TestNoopLogger_UsageExample(t *testing.T) {
	// Create logger for test
	logger := NewNoopLogger()

	// Simulate some function that uses logging
	simulateFunction := func() {
		logger.Info("Starting operation")
		logger.Debug("Processing item %d", 1)
		logger.Warn("Low memory warning")
		logger.Info("Operation completed successfully")
	}

	// Execute the function
	simulateFunction()

	// Assert expected log messages
	assert.Equal(t, 4, logger.Len())
	assert.True(t, logger.ContainsLevel("INFO", "Starting operation"))
	assert.True(t, logger.ContainsLevel("DEBUG", "Processing item 1"))
	assert.True(t, logger.ContainsLevel("WARN", "Low memory warning"))
	assert.True(t, logger.ContainsLevel("INFO", "Operation completed"))

	// Verify message order
	messages := logger.GetMessages()
	assert.Equal(t, "Starting operation", messages[0])
	assert.Equal(t, "Processing item 1", messages[1])
	assert.Equal(t, "Low memory warning", messages[2])
	assert.Equal(t, "Operation completed successfully", messages[3])

	// Test specific level filtering
	infoMessages := logger.GetMessagesByLevel("INFO")
	assert.Len(t, infoMessages, 2)
	assert.Contains(t, infoMessages, "Starting operation")
	assert.Contains(t, infoMessages, "Operation completed successfully")
}
