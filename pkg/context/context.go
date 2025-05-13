package context

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// WithShutdown creates a new context that will be cancelled on SIGTERM/SIGINT
func WithShutdown(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		fmt.Printf("caught interrupt, shutting down gracefully after %d second(s)\n", 1*time.Second)
		time.Sleep(1 * time.Second)
		cancel()
	}()

	return ctx
}
