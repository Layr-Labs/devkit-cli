package iface

// Actor represents different actors in the system for color-coded logging
type Actor string

const (
	ActorSystem    Actor = "SYSTEM"    // System operations, Docker, file I/O, Git operations
	ActorOperator  Actor = "OPERATOR"  // Operator related operations, keystore, registration
	ActorAVSDev    Actor = "AVS_DEV"   // AVS developer operations, building, running, creating projects
	ActorConfig    Actor = "CONFIG"    // Configuration and context management
	ActorTelemetry Actor = "TELEMETRY" // Telemetry related operations
)

type Logger interface {
	Title(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)

	// Actor-based methods for color-coded logging
	TitleWithActor(actor Actor, msg string, args ...any)
	InfoWithActor(actor Actor, msg string, args ...any)
	WarnWithActor(actor Actor, msg string, args ...any)
	ErrorWithActor(actor Actor, msg string, args ...any)
	DebugWithActor(actor Actor, msg string, args ...any)
}

type ProgressLogger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Progress(name string, percent int, displayText string)
	PrintProgress()
	ClearProgress()
}
