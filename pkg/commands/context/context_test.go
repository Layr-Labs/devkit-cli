package context

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

func setupCLIContext(cmd *cli.Command, args []string, flags map[string]string) *cli.Context {
	fs := flag.NewFlagSet("test", 0)
	for _, f := range cmd.Flags {
		if err := f.Apply(fs); err != nil {
			panic(err)
		}
	}
	// build args slice: first positional args, then flag pairs
	var argv []string
	argv = append(argv, args...)
	for k, v := range flags {
		argv = append(argv, "--"+k, v)
	}
	if err := fs.Parse(argv); err != nil {
		panic(err)
	}

	// Create app with no-op logger and progress tracker
	noopLogger := logger.NewNoopLogger()
	noopProgressTracker := logger.NewNoopProgressTracker()
	app := &cli.App{
		Before: func(cCtx *cli.Context) error {
			ctx := common.WithLogger(cCtx.Context, noopLogger)
			ctx = common.WithProgressTracker(ctx, noopProgressTracker)
			cCtx.Context = ctx
			return nil
		},
	}

	ctx := cli.NewContext(app, fs, nil)

	// Execute the Before hook to set up the logger context
	if app.Before != nil {
		if err := app.Before(ctx); err != nil {
			panic(err)
		}
	}

	return ctx
}

func TestNewModelDefaults(t *testing.T) {
	m := NewModel("label", []string{"a", "b"})
	require.Equal(t, "label", m.Label)
	require.Equal(t, []string{"a", "b"}, m.Choices)
}

func TestCreateContextFunction(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "foo.yaml")
	err := CreateContext(path, "foo")
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "foo")
}

func TestCreateContextCommand_CreatesFile(t *testing.T) {
	tmp := t.TempDir()
	ctx := setupCLIContext(CreateContextCommand, nil, map[string]string{"context": "bar"})

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if ch := os.Chdir(orig); ch != nil {
			t.Fatal(ch)
		}
	}()

	require.NoError(t, os.Chdir(tmp))

	err = CreateContextCommand.Action(ctx)
	require.NoError(t, err)

	want := filepath.Join("config", "contexts", "bar.yaml")
	_, statErr := os.Stat(want)
	require.NoError(t, statErr)
}

func TestListContexts_NoDir(t *testing.T) {
	tmp := t.TempDir()
	_, err := ListContexts(filepath.Join(tmp, "nodir"), true)
	require.Error(t, err)
}

func TestListContexts_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	require.NoError(t, os.MkdirAll(tmp, 0755))

	orig := RunSelection
	defer func() { RunSelection = orig }()
	RunSelection = func(label string, opts []string) (string, error) {
		require.Empty(t, opts)
		return "", fmt.Errorf("no opts")
	}

	_, err := ListContexts(tmp, false)
	require.Error(t, err)
}

func TestListContexts_Success(t *testing.T) {
	tmp := t.TempDir()
	ctxDir := filepath.Join(tmp, "contexts")
	require.NoError(t, os.MkdirAll(ctxDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(ctxDir, "foo.yaml"), []byte("dummy"), 0644))

	orig := RunSelection
	defer func() { RunSelection = orig }()
	RunSelection = func(label string, opts []string) (string, error) {
		require.Equal(t, "Which context would you like to list?", label)
		require.Equal(t, []string{"foo"}, opts)
		return "foo", nil
	}

	got, err := ListContexts(ctxDir, true)
	require.NoError(t, err)
	require.Equal(t, []string{"foo"}, got)
}

func TestSetFlagWritesYAML(t *testing.T) {
	// prepare temp config/contexts/test.yaml
	tmp := t.TempDir()
	base := filepath.Join(tmp, "config", "contexts")
	require.NoError(t, os.MkdirAll(base, 0755))
	yml := `
context:
  project:
    name: old
`
	filePath := filepath.Join(base, "test.yaml")
	require.NoError(t, os.WriteFile(filePath, []byte(yml), 0644))

	// set up CLI with --context test and --set project.name=new
	ctx := setupCLIContext(Command, nil, map[string]string{
		"context": "test",
		"set":     "project.name=new",
	})

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if ch := os.Chdir(orig); ch != nil {
			t.Fatal(ch)
		}
	}()

	require.NoError(t, os.Chdir(tmp))

	// execute
	err = Command.Action(ctx)
	require.NoError(t, err)

	// verify YAML updated
	out, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal(out, &doc))

	// navigate to context.project.name
	root := doc.Content[0]
	var found string
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == "context" {
			mapNode := root.Content[i+1]
			for j := 0; j < len(mapNode.Content); j += 2 {
				if mapNode.Content[j].Value == "project" {
					prj := mapNode.Content[j+1]
					for k := 0; k < len(prj.Content); k += 2 {
						if prj.Content[k].Value == "name" {
							found = prj.Content[k+1].Value
						}
					}
				}
			}
		}
	}
	require.Equal(t, "new", found)
}

func TestContextSetsGlobalContext(t *testing.T) {
	// prepare temp config/config.yaml and a dummy context file
	tmp := t.TempDir()
	// config.yaml
	cfgDir := filepath.Join(tmp, "config")
	require.NoError(t, os.MkdirAll(cfgDir, 0755))
	baseYML := `
config:
  project:
    name: demo
`
	cfgPath := filepath.Join(cfgDir, common.BaseConfig)
	require.NoError(t, os.WriteFile(cfgPath, []byte(baseYML), 0644))

	// contexts/prod.yaml (must exist so Action doesn't error)
	contextsDir := filepath.Join(cfgDir, "contexts")
	require.NoError(t, os.MkdirAll(contextsDir, 0755))
	prodCtx := filepath.Join(contextsDir, "prod.yaml")
	require.NoError(t, os.WriteFile(prodCtx, []byte("context: {}"), 0644))

	// set up CLI with --context prod (no --list)
	ctx := setupCLIContext(Command, nil, map[string]string{
		"context": "prod",
	})

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if ch := os.Chdir(orig); ch != nil {
			t.Fatal(ch)
		}
	}()

	require.NoError(t, os.Chdir(tmp))

	// execute
	err = Command.Action(ctx)
	require.NoError(t, err)

	// verify config.yaml has project.context=prod
	out, err := os.ReadFile(cfgPath)
	require.NoError(t, err)

	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal(out, &doc))

	root := doc.Content[0]
	var found string
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == "config" {
			cfgNode := root.Content[i+1]
			for j := 0; j < len(cfgNode.Content); j += 2 {
				if cfgNode.Content[j].Value == "project" {
					prj := cfgNode.Content[j+1]
					for k := 0; k < len(prj.Content); k += 2 {
						if prj.Content[k].Value == "context" {
							found = prj.Content[k+1].Value
						}
					}
				}
			}
		}
	}
	require.Equal(t, "prod", found)
}

func TestSettingNonexistentContextFails(t *testing.T) {
	// Prepare a temp project with a valid config.yaml but no contexts/foo.yaml
	tmp := t.TempDir()
	// Create config/config.yaml
	cfgDir := filepath.Join(tmp, "config")
	require.NoError(t, os.MkdirAll(cfgDir, 0755))
	baseYML := `
config:
  project:
    name: demo
`
	cfgPath := filepath.Join(cfgDir, common.BaseConfig)
	require.NoError(t, os.WriteFile(cfgPath, []byte(baseYML), 0644))
	// Create an empty contexts directory
	contextsDir := filepath.Join(tmp, "config", "contexts")
	require.NoError(t, os.MkdirAll(contextsDir, 0755))

	// Build CLI context for Command with --context=foo (which does not exist)
	ctx := setupCLIContext(Command, nil, map[string]string{
		"context": "foo",
	})

	// Switch into tmp so relative paths resolve
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if ch := os.Chdir(orig); ch != nil {
			t.Fatal(ch)
		}
	}()

	require.NoError(t, os.Chdir(tmp))

	// Execute
	err = Command.Action(ctx)
	require.Error(t, err)
	// Check the error message suggests creating the context
	require.Contains(t, err.Error(),
		"this context does not exist, create it with `devkit avs context create foo`")
}
