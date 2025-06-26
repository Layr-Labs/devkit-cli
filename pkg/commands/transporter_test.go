package commands

import (
	"context"
	"testing"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestScheduleTransport_InvalidCron(t *testing.T) {
	app := cli.NewApp()
	cCtx := cli.NewContext(app, nil, nil)
	cCtx.Context = context.Background()

	// attempt to schedule with invalid cron expression
	err := ScheduleTransport(cCtx, "invalid-cron")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid cron expression")
}

func TestScheduleTransportWithParserAndFunc_Executes(t *testing.T) {
	executed := make(chan struct{}, 1)
	mockFunc := func() {
		executed <- struct{}{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := cli.NewApp()
	cCtx := cli.NewContext(app, nil, nil)
	cCtx.Context = ctx

	// pass in second aware parser
	cronExpr := "*/1 * * * * *"
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	go func() {
		err := ScheduleTransportWithParserAndFunc(cCtx, cronExpr, parser, mockFunc)
		require.NoError(t, err)
	}()

	select {
	case <-executed:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("transportFunc did not execute as expected")
	}
}

func TestScheduleTransportWithParserAndFunc_ContextCancellationStopsScheduler(t *testing.T) {
	executed := make(chan struct{}, 1)
	mockFunc := func() {
		executed <- struct{}{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	app := cli.NewApp()
	cCtx := cli.NewContext(app, nil, nil)
	cCtx.Context = ctx

	// pass in minute aware parser
	cronExpr := "*/1 * * * *"
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	go func() {
		_ = ScheduleTransportWithParserAndFunc(cCtx, cronExpr, parser, mockFunc)
	}()

	// cancel early to test shutdown
	time.Sleep(500 * time.Millisecond)
	cancel()

	// wait and confirm function was not called
	select {
	case <-executed:
		t.Fatal("transportFunc should not have executed after early cancel")
	case <-time.After(2 * time.Second):
		// success
	}
}
