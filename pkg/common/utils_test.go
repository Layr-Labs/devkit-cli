package common

import (
	"reflect"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestIsVerboseEnabled(t *testing.T) {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "verbose"},
		},
		Action: func(cCtx *cli.Context) error {
			cfg := &ConfigWithContextConfig{}
			if !IsVerboseEnabled(cCtx, cfg) {
				t.Errorf("expected true when verbose flag is set")
			}
			return nil
		},
	}

	err := app.Run([]string{"test", "--verbose"})
	if err != nil {
		t.Fatalf("cli run failed: %v", err)
	}
}

func TestGetLogger_ReturnsLoggerAndTracker(t *testing.T) {
	log, tracker := GetLogger()

	logType := reflect.TypeOf(log).String()
	trackerType := reflect.TypeOf(tracker).String()

	if !isValidLogger(logType) {
		t.Errorf("unexpected logger type: %s", logType)
	}
	if !isValidTracker(trackerType) {
		t.Errorf("unexpected tracker type: %s", trackerType)
	}
}

func isValidLogger(typ string) bool {
	return typ == "*logger.Logger" || typ == "*logger.ZapLogger"
}

func isValidTracker(typ string) bool {
	return typ == "*progress.TTYProgressTracker" || typ == "*progress.LogProgressTracker"
}
