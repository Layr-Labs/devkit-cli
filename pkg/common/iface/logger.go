package iface

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)

	InfoWithActor(actor string, msg string, args ...any)
	WarnWithActor(actor string, msg string, args ...any)
	ErrorWithActor(actor string, msg string, args ...any)
}

type ProgressLogger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Progress(name string, percent int, displayText string)
	PrintProgress()
	ClearProgress()
}
