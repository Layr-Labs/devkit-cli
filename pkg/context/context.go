package context

import (
	"context"
	"devkit-cli/pkg/common/logger"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// Embedded devkit version from release
var embeddedDevkitReleaseVersion = "dev"

// WithShutdown creates a new context that will be cancelled on SIGTERM/SIGINT
func WithShutdown(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		signal.Stop(sigChan)
		cancel()
		_, _ = fmt.Fprintln(os.Stderr, "caught interrupt, shutting down gracefully.")
	}()

	return ctx
}

type appEnvironmentContextKey struct{}

type AppEnvironment struct {
	CLIVersion  string
	OS          string
	Arch        string
	ProjectUUID string
}

func NewAppEnvironment(os string, arch string, projectUuid string) *AppEnvironment {
	logger.NewLogger().Info("embedded application release embeddedDevkitReleaseVersion: %s", embeddedDevkitReleaseVersion)
	return &AppEnvironment{
		CLIVersion:  embeddedDevkitReleaseVersion,
		OS:          os,
		Arch:        arch,
		ProjectUUID: projectUuid,
	}
}

func WithAppEnvironment(ctx context.Context, appEnvironment *AppEnvironment) context.Context {
	return context.WithValue(ctx, appEnvironmentContextKey{}, appEnvironment)
}

func AppEnvironmentFromContext(ctx context.Context) (*AppEnvironment, bool) {
	env, ok := ctx.Value(appEnvironmentContextKey{}).(*AppEnvironment)
	return env, ok
}
