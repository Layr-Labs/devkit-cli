package keystore

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/testutils"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestKeystoreCreateAndRead(t *testing.T) {
	tmpDir := t.TempDir()

	key := "12248929636257230549931416853095037629726205319386239410403476017439825112537"
	password := "testpass"
	path := filepath.Join(tmpDir, "operator1.keystore.json")

	// Create keystore with no-op logger
	createCmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(CreateCommand)
	app := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name:        "keystore",
				Subcommands: []*cli.Command{createCmdWithLogger},
				Before: func(cCtx *cli.Context) error {
					// Execute the subcommand's Before hook to set up logger context
					if createCmdWithLogger.Before != nil {
						return createCmdWithLogger.Before(cCtx)
					}
					return nil
				},
			},
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

	// Read keystore with no-op logger
	readCmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(ReadCommand)
	readApp := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name:        "keystore",
				Subcommands: []*cli.Command{readCmdWithLogger},
				Before: func(cCtx *cli.Context) error {
					// Execute the subcommand's Before hook to set up logger context
					if readCmdWithLogger.Before != nil {
						return readCmdWithLogger.Before(cCtx)
					}
					return nil
				},
			},
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

func TestECDSAKeystoreCreateAndRead(t *testing.T) {
	tmpDir := t.TempDir()

	// Test with 0x prefix
	key := "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
	password := "testpass"
	path := filepath.Join(tmpDir, "operator1.ecdsa.keystore.json")

	// Create ECDSA keystore
	createCmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(CreateCommand)
	app := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name:        "keystore",
				Subcommands: []*cli.Command{createCmdWithLogger},
				Before: func(cCtx *cli.Context) error {
					if createCmdWithLogger.Before != nil {
						return createCmdWithLogger.Before(cCtx)
					}
					return nil
				},
			},
		},
	}
	err := app.Run([]string{
		"devkit", "keystore", "create",
		"--key", key,
		"--path", path,
		"--type", "ecdsa",
		"--password", password,
	})
	require.NoError(t, err)

	// Verify keystore file was created
	_, err = os.Stat(path)
	require.NoError(t, err, "expected ECDSA keystore file to be created")

	// Read keystore with no-op logger
	readCmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(ReadCommand)
	readApp := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name:        "keystore",
				Subcommands: []*cli.Command{readCmdWithLogger},
				Before: func(cCtx *cli.Context) error {
					if readCmdWithLogger.Before != nil {
						return readCmdWithLogger.Before(cCtx)
					}
					return nil
				},
			},
		},
	}

	// Capture logs via pipe
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
	log.SetOutput(os.Stderr)

	// Read from pipe
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	output := buf.String()
	require.Contains(t, output, "Save this ECDSA private key in a secure location")
	require.Contains(t, output, key)
}

func TestECDSAKeystoreCreateWithoutPrefix(t *testing.T) {
	tmpDir := t.TempDir()

	// Test without 0x prefix
	key := "7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
	password := "securepass"
	path := filepath.Join(tmpDir, "operator2.ecdsa.keystore.json")

	// Create ECDSA keystore
	createCmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(CreateCommand)
	app := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name:        "keystore",
				Subcommands: []*cli.Command{createCmdWithLogger},
				Before: func(cCtx *cli.Context) error {
					if createCmdWithLogger.Before != nil {
						return createCmdWithLogger.Before(cCtx)
					}
					return nil
				},
			},
		},
	}
	err := app.Run([]string{
		"devkit", "keystore", "create",
		"--key", key,
		"--path", path,
		"--type", "ecdsa",
		"--password", password,
	})
	require.NoError(t, err)

	// Verify keystore file was created
	_, err = os.Stat(path)
	require.NoError(t, err, "expected ECDSA keystore file to be created")

	// Verify it's a valid JSON file
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	var keystore map[string]interface{}
	err = json.Unmarshal(content, &keystore)
	require.NoError(t, err)

	// Check it has the expected fields
	require.Contains(t, keystore, "address")
	require.Contains(t, keystore, "crypto")
	require.Contains(t, keystore, "version")
}

func TestKeystoreCreateInvalidType(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.keystore.json")

	createCmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(CreateCommand)
	app := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name:        "keystore",
				Subcommands: []*cli.Command{createCmdWithLogger},
				Before: func(cCtx *cli.Context) error {
					if createCmdWithLogger.Before != nil {
						return createCmdWithLogger.Before(cCtx)
					}
					return nil
				},
			},
		},
	}
	err := app.Run([]string{
		"devkit", "keystore", "create",
		"--key", "somekey",
		"--path", path,
		"--type", "invalid",
		"--password", "pass",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported curve type")
}

func TestECDSAKeystoreWrongPassword(t *testing.T) {
	tmpDir := t.TempDir()

	key := "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
	password := "correctpass"
	wrongPassword := "wrongpass"
	path := filepath.Join(tmpDir, "operator.ecdsa.keystore.json")

	// Create ECDSA keystore
	createCmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(CreateCommand)
	app := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name:        "keystore",
				Subcommands: []*cli.Command{createCmdWithLogger},
				Before: func(cCtx *cli.Context) error {
					if createCmdWithLogger.Before != nil {
						return createCmdWithLogger.Before(cCtx)
					}
					return nil
				},
			},
		},
	}
	err := app.Run([]string{
		"devkit", "keystore", "create",
		"--key", key,
		"--path", path,
		"--type", "ecdsa",
		"--password", password,
	})
	require.NoError(t, err)

	// Try to read with wrong password
	readCmdWithLogger, _ := testutils.WithTestConfigAndNoopLoggerAndAccess(ReadCommand)
	readApp := &cli.App{
		Name: "devkit",
		Commands: []*cli.Command{
			{
				Name:        "keystore",
				Subcommands: []*cli.Command{readCmdWithLogger},
				Before: func(cCtx *cli.Context) error {
					if readCmdWithLogger.Before != nil {
						return readCmdWithLogger.Before(cCtx)
					}
					return nil
				},
			},
		},
	}

	readArgs := []string{
		"devkit", "keystore", "read",
		"--path", path,
		"--password", wrongPassword,
	}
	err = readApp.Run(readArgs)
	require.Error(t, err)
	require.Contains(t, err.Error(), "could not decrypt")
}
