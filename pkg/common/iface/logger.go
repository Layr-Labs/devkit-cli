package iface

type Logger interface {
	Title(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
}

type ProgressLogger interface {
	Title(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Progress(name string, percent int, displayText string)
	PrintProgress()
	ClearProgress()
}
