package keystore

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestKeystoreCreateAndRead(t *testing.T) {
	tmpDir := t.TempDir()

	key := "12248929636257230549931416853095037629726205319386239410403476017439825112537"
	password := "testpass"
	path := filepath.Join(tmpDir, "operator1.keystore.json")

	// --- Run create keystore ---
	app := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name: "keystore",
				Subcommands: []*cli.Command{
					CreateCommand,
				},
			},
		},
	}

	args := []string{
		"devkit", "keystore", "create",
		"--key", key,
		"--path", path,
		"--type", "bn254",
		"--password", password,
	}

	err := app.Run(args)
	require.NoError(t, err)

	// 🔒 Verify keystore file was created
	_, err = os.Stat(path)
	require.NoError(t, err, "expected keystore file to be created")

	// --- Run read keystore ---
	readApp := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name: "keystore",
				Subcommands: []*cli.Command{
					ReadCommand,
				},
			},
		},
	}

	// 🧪 Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	readArgs := []string{
		"devkit", "keystore", "read",
		"--path", path,
		"--password", password,
	}
	err = readApp.Run(readArgs)
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	output := buf.String()
	require.Contains(t, output, "Save this BLS private key", "expected private key output in read command")
	require.Contains(t, output, key, "expected same private key as input")
}