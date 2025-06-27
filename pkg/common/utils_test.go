package common

import (
	"math/big"
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
	log, tracker := GetLogger(false)

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

func TestParseETHAmount(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // Expected result in wei as string
		wantErr  bool
	}{
		{
			name:     "Simple ETH amount",
			input:    "5ETH",
			expected: "5000000000000000000", // 5 * 10^18
			wantErr:  false,
		},
		{
			name:     "Decimal ETH amount",
			input:    "1.5ETH",
			expected: "1500000000000000000", // 1.5 * 10^18
			wantErr:  false,
		},
		{
			name:     "Case insensitive ETH",
			input:    "10eth",
			expected: "10000000000000000000", // 10 * 10^18
			wantErr:  false,
		},
		{
			name:     "ETH with spaces",
			input:    "  2.5 ETH  ",
			expected: "2500000000000000000", // 2.5 * 10^18
			wantErr:  false,
		},
		{
			name:     "Wei amount (no ETH suffix)",
			input:    "1000000000000000000",
			expected: "1000000000000000000", // 1 * 10^18
			wantErr:  false,
		},
		{
			name:     "Zero ETH",
			input:    "0ETH",
			expected: "0",
			wantErr:  false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Invalid ETH amount",
			input:    "invalidETH",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "Invalid wei amount",
			input:    "invalid123",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseETHAmount(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseETHAmount() expected error for input '%s', but got none", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseETHAmount() unexpected error for input '%s': %v", tt.input, err)
				return
			}

			expected := new(big.Int)
			expected.SetString(tt.expected, 10)

			if result.Cmp(expected) != 0 {
				t.Errorf("ParseETHAmount() for input '%s' = %s, expected %s", tt.input, result.String(), expected.String())
			}
		})
	}
}
