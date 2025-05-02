package keystore

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestKeystoreCreateAndRead(t *testing.T) {
	tmpDir := t.TempDir()

	key := "12248929636257230549931416853095037629726205319386239410403476017439825112537"
	password := "testpass"
	path := filepath.Join(tmpDir, "operator1.keystore.json")

	// Create keystore
	app := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{Name: "keystore", Subcommands: []*cli.Command{CreateCommand}},
		},
	}
	err := app.Run([]string{
		"devkit", "keystore", "create",
		"--key", key,
		"--path", path,
		"--type", "bn254",
		"--password", password,
	})
	require.NoError(t, err)

	// 🔒 Verify keystore file was created
	_, err = os.Stat(path)
	require.NoError(t, err, "expected keystore file to be created")

	// Read keystore
	readApp := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{Name: "keystore", Subcommands: []*cli.Command{ReadCommand}},
		},
	}

	// 🧪 Capture logs via pipe
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdout = w
	os.Stderr = w
	log.SetOutput(w)

	readArgs := []string{
		"devkit", "keystore", "read",
		"--path", path,
		"--password", password,
	}
	err = readApp.Run(readArgs)
	require.NoError(t, err)

	// Close writer and restore
	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	log.SetOutput(os.Stderr) // Restore default log output

	// Read from pipe
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	output := buf.String()
	require.Contains(t, output, "Save this BLS private key in a secure location")
	require.Contains(t, output, key)
}
