package logger

import (
	"fmt"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
)

// ANSI color codes
const (
	ColorReset  = "\033[0m"
	ColorBlue   = "\033[34m" // SYSTEM
	ColorGreen  = "\033[32m" // OPERATOR
	ColorCyan   = "\033[36m" // AVS_DEV
	ColorYellow = "\033[33m" // CONFIG
	ColorPurple = "\033[35m" // TELEMETRY
	ColorRed    = "\033[31m" // ERROR
	ColorOrange = "\033[93m" // WARN
	ColorGray   = "\033[90m" // DEBUG
	ColorBold   = "\033[1m"
)

// ColoredLogger wraps an existing logger and adds color-coded actor-based logging
type ColoredLogger struct {
	base iface.Logger
}

// NewColoredLogger creates a new colored logger that wraps the provided base logger
func NewColoredLogger(base iface.Logger) *ColoredLogger {
	return &ColoredLogger{
		base: base,
	}
}

// getActorColor returns the color code for the given actor
func (c *ColoredLogger) getActorColor(actor iface.Actor) string {
	switch actor {
	case iface.ActorSystem:
		return ColorBlue
	case iface.ActorOperator:
		return ColorGreen
	case iface.ActorAVSDev:
		return ColorCyan
	case iface.ActorConfig:
		return ColorYellow
	case iface.ActorTelemetry:
		return ColorPurple
	default:
		return ColorReset
	}
}

// formatMessage formats a message with actor color and label
func (c *ColoredLogger) formatMessage(actor iface.Actor, level string, msg string, args ...any) string {
	actorColor := c.getActorColor(actor)

	// Format the message with args
	formatted := fmt.Sprintf(msg, args...)

	// Add actor label with color
	actorLabel := fmt.Sprintf("%s[%s]%s", actorColor, string(actor), ColorReset)

	// Add level color if applicable
	var levelColor string
	switch level {
	case "ERROR":
		levelColor = ColorRed
	case "WARN":
		levelColor = ColorOrange
	case "DEBUG":
		levelColor = ColorGray
	case "TITLE":
		levelColor = ColorBold
	default:
		levelColor = ""
	}

	if levelColor != "" {
		if level == "TITLE" {
			return fmt.Sprintf("%s %s%s%s", actorLabel, levelColor, formatted, ColorReset)
		} else {
			return fmt.Sprintf("%s %s[%s]%s %s", actorLabel, levelColor, level, ColorReset, formatted)
		}
	}

	return fmt.Sprintf("%s %s", actorLabel, formatted)
}

// Traditional methods (for backward compatibility)
func (c *ColoredLogger) Title(msg string, args ...any) {
	c.base.Title(msg, args...)
}

func (c *ColoredLogger) Info(msg string, args ...any) {
	c.base.Info(msg, args...)
}

func (c *ColoredLogger) Warn(msg string, args ...any) {
	c.base.Warn(msg, args...)
}

func (c *ColoredLogger) Error(msg string, args ...any) {
	c.base.Error(msg, args...)
}

func (c *ColoredLogger) Debug(msg string, args ...any) {
	c.base.Debug(msg, args...)
}

// Actor-based colored methods
func (c *ColoredLogger) TitleWithActor(actor iface.Actor, msg string, args ...any) {
	formatted := c.formatMessage(actor, "TITLE", msg, args...)

	// For titles, we want to add the newlines like the original Title method
	lines := strings.Split("\n"+formatted+"\n", "\n")
	for _, line := range lines {
		c.base.Info("%s", line)
	}
}

func (c *ColoredLogger) InfoWithActor(actor iface.Actor, msg string, args ...any) {
	formatted := c.formatMessage(actor, "INFO", msg, args...)
	c.base.Info("%s", formatted)
}

func (c *ColoredLogger) WarnWithActor(actor iface.Actor, msg string, args ...any) {
	formatted := c.formatMessage(actor, "WARN", msg, args...)
	c.base.Info("%s", formatted)
}

func (c *ColoredLogger) ErrorWithActor(actor iface.Actor, msg string, args ...any) {
	formatted := c.formatMessage(actor, "ERROR", msg, args...)
	c.base.Info("%s", formatted)
}

func (c *ColoredLogger) DebugWithActor(actor iface.Actor, msg string, args ...any) {
	formatted := c.formatMessage(actor, "DEBUG", msg, args...)
	c.base.Info("%s", formatted)
}
