package commands

import (
	"bytes"
	"devkit-cli/pkg/common"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestConfigCommand_ListOutput(t *testing.T) {
	tmpDir := t.TempDir()

	defaultConfigPath := filepath.Join("..", "..", "config")
	defaultConfigFile, err := os.ReadFile(filepath.Join(defaultConfigPath, common.BaseConfig))
	require.NoError(t, err)

	defaultDevnetConfigFile, err := os.ReadFile(filepath.Join(defaultConfigPath, "contexts", "devnet.yaml"))
	require.NoError(t, err)
	configPath := filepath.Join(tmpDir, "config")
	require.NoError(t, os.MkdirAll(configPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configPath, common.BaseConfig), defaultConfigFile, 0644))
	contextsPath := filepath.Join(configPath, "contexts")
	require.NoError(t, os.MkdirAll(contextsPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(contextsPath, "devnet.yaml"), defaultDevnetConfigFile, 0644))

	mockTelemteryContent :=
		`
project_uuid: d7598c91-2ec4-4751-b0ab-bc848f73d58e
telemetry_enabled: true
`

	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, common.DevkitConfigFile),
		[]byte(mockTelemteryContent),
		0644,
	))
	// 🔁 Change into the test directory
	originalWD, _ := os.Getwd()
	defer func() {
		if err := os.Chdir(originalWD); err != nil {
			t.Logf("Failed to return to original directory: %v", err)
		}
	}()
	require.NoError(t, os.Chdir(tmpDir))

	// 🧪 Capture os.Stdout
	var buf bytes.Buffer
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// ⚙️ Run the CLI app with nested subcommands
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name: "avs",
				Subcommands: []*cli.Command{
					ConfigCommand,
				},
			},
		},
	}
	err = app.Run([]string{"devkit", "avs", "config", "--list"})
	require.NoError(t, err)

	// 📤 Finish capturing output
	w.Close()
	os.Stdout = stdout
	_, _ = buf.ReadFrom(r)
	// output := stripANSI(buf.String())

	// ✅ Validating output
	// require.Contains(t, output, "[project]")
	// require.Contains(t, output, "[operator]")
	// require.Contains(t, output, "[env]")
}
