package common

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

// Embedded devkit version from release
var embeddedDevkitReleaseVersion = "Development"

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
	UserUUID    string
}

func NewAppEnvironment(os, arch, projectUuid, userUuid string) *AppEnvironment {
	return &AppEnvironment{
		CLIVersion:  embeddedDevkitReleaseVersion,
		OS:          os,
		Arch:        arch,
		ProjectUUID: projectUuid,
		UserUUID:    userUuid,
	}
}

func WithAppEnvironment(ctx *cli.Context) {
	withAppEnvironmentFromLocation(ctx, filepath.Join("config", "config.yaml"))
}

func withAppEnvironmentFromLocation(ctx *cli.Context, location string) {
	user := getUserUUIDFromGlobalConfig()
	if user == "" {
		user = uuid.New().String()
	}

	id := getProjectUUIDFromLocation(location)
	if id == "" {
		id = uuid.New().String()
	}
	ctx.Context = withAppEnvironment(ctx.Context, NewAppEnvironment(
		runtime.GOOS,
		runtime.GOARCH,
		id,
		user,
	))
}

func withAppEnvironment(ctx context.Context, appEnvironment *AppEnvironment) context.Context {
	return context.WithValue(ctx, appEnvironmentContextKey{}, appEnvironment)
}

func AppEnvironmentFromContext(ctx context.Context) (*AppEnvironment, bool) {
	env, ok := ctx.Value(appEnvironmentContextKey{}).(*AppEnvironment)
	return env, ok
}
