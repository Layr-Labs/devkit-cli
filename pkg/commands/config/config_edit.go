package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/Layr-Labs/devkit-cli/pkg/telemetry"
	"go.uber.org/zap"

	"github.com/urfave/cli/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"sigs.k8s.io/yaml"
)

// ConfigChange represents a change in a configuration field
type ConfigChange struct {
	Path     string
	OldValue interface{}
	NewValue interface{}
}

// Iota based enum for types of config
type EditTarget int

const (
	Config EditTarget = iota
	Context
)

// Path to the config directory
var DefaultConfigPath = filepath.Join("config")

// editConfig is the main entry point for the edit config functionality
func EditConfig(cCtx *cli.Context, configPath string, editTarget EditTarget, context string) error {
	logger := common.LoggerFromContext(cCtx.Context)

	// Find an available editor
	editor, err := findEditor()
	if err != nil {
		return err
	}

	// Create a backup of the current config
	_, backupData, err := backupConfig(configPath, editTarget, context)
	if err != nil {
		return err
	}

	// Open the editor and wait for it to close
	if err := openEditor(editor, configPath, logger); err != nil {
		return err
	}

	// Validate the edited config
	newData, err := ValidateConfig(configPath, editTarget)
	// Check for validation errs
	if err != nil {
		logger.Error("Error validating config: %v", err)
		logger.Info("Reverting changes...")
		if restoreErr := restoreBackup(configPath, backupData); restoreErr != nil {
			logger.Error("Failed to restore backup after validation error: %w", restoreErr)
			return restoreErr
		}
		return err
	}

	// Collect changes
	changes, err := validateConfigChanges(backupData, newData)
	if err != nil {
		logger.Error("Error validating config: %v", err)
		logger.Info("Reverting changes...")
		if restoreErr := restoreBackup(configPath, backupData); restoreErr != nil {
			logger.Error("Failed to restore backup after validation error: %w", restoreErr)
			return restoreErr
		}
		return err
	}

	// Log changes
	logConfigChanges(changes, logger)

	// Send telemetry
	sendConfigChangeTelemetry(cCtx.Context, changes, logger)

	logger.Info("Config file updated successfully.")
	return nil
}

// findEditor looks for available text editors
func findEditor() (string, error) {
	// Try to use the EDITOR environment variable
	if editor := os.Getenv("EDITOR"); editor != "" {
		if _, err := exec.LookPath(editor); err == nil {
			return editor, nil
		}
	}

	// Try common editors in order of preference
	for _, editor := range []string{"nano", "vi", "vim"} {
		if path, err := exec.LookPath(editor); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no suitable text editor found. Please install nano or vi, or set the EDITOR environment variable")
}

// backupConfig creates a backup of the current config
func backupConfig(configPath string, editTarget EditTarget, context string) (map[string]interface{}, []byte, error) {
	// Load the current config to compare later
	var (
		currentConfig map[string]interface{}
		err           error
	)

	// Select the interface based on target
	if editTarget == Config {
		currentConfig, err = common.LoadBaseConfig()
	} else if editTarget == Context {
		currentConfig, err = common.LoadContextConfig(context)
	} else {
		return nil, nil, fmt.Errorf("error selecting yaml: %w", err)
	}

	// If there is any error loading the yaml
	if err != nil {
		return nil, nil, fmt.Errorf("error loading yaml: %w", err)
	}

	// Read the raw file data
	file, err := os.Open(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening yaml file: %w", err)
	}
	defer file.Close()

	backupData, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading config file: %w", err)
	}

	return currentConfig, backupData, nil

}

// openEditor launches the editor for the config file
func openEditor(editorPath, filePath string, logger iface.Logger) error {
	logger.Info("Opening config file in %s...", editorPath)

	cmd := exec.Command(editorPath, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ValidateConfig reads configPath into the appropriate struct based on editTarget,
// then runs requireNonZero on it, returning the raw bytes or an error.
func ValidateConfig(configPath string, editTarget EditTarget) ([]byte, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Either Config or ContextConfig
	var val interface{}

	// Perform validations on the provided struct
	if editTarget == Config {
		// Try unmarshalling as BaseConfig (config.yaml)
		var cfg common.Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return data, fmt.Errorf("invalid base config YAML: %w", err)
		}
		val = &cfg

	} else if editTarget == Context {
		// Try unmarshalling as ContextConfig (devnet.yaml, sepolia.yaml)
		var ctx common.ContextConfig
		if err := yaml.Unmarshal(data, &ctx); err != nil {
			return data, fmt.Errorf("invalid context config YAML: %w", err)
		}
		val = &ctx

	} else {
		return data, fmt.Errorf("unsupported edit target: %v", editTarget)
	}

	// Verify known fields are present
	if err := common.RequireNonZero(val); err != nil {
		return data, err
	}

	return data, nil
}

// restoreBackup restores the original file content
func restoreBackup(configPath string, backupData []byte) error {
	return os.WriteFile(configPath, backupData, 0644)
}

// validateConfigChanges returns adds/removes/changes between two generic maps
func validateConfigChanges(
	originalYAML, updatedYAML []byte,
) ([]ConfigChange, error) {
	original, err := common.YamlToMap(originalYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to convert yaml to map: %w", err)
	}
	updated, err := common.YamlToMap(updatedYAML)
	if err != nil {
		return nil, fmt.Errorf("failed to convert yaml to map: %w", err)
	}

	// Ensure version string is present and unchanged
	ov, ok := original["version"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or non-string 'version' in original")
	}
	nv, ok := updated["version"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or non-string 'version' in updated")
	}
	if ov != nv {
		return nil, fmt.Errorf("version must not be altered (was %q, now %q)", ov, nv)
	}

	changes := diffValues("", original, updated)
	return changes, nil
}

// diffValues recurses into maps, slices and primitives.
func diffValues(path string, oldV, newV interface{}) []ConfigChange {
	var out []ConfigChange

	// If oldV is map[interface{}]interface{}, turn into map[string]interface{}
	if m, ok := oldV.(map[interface{}]interface{}); ok {
		oldV = common.Normalize(m)
	}
	if m, ok := newV.(map[interface{}]interface{}); ok {
		newV = common.Normalize(m)
	}

	// nil/absent cases
	if oldV == nil && newV == nil {
		return nil
	}
	if oldV == nil {
		out = append(out, ConfigChange{Path: path, OldValue: nil, NewValue: newV})
		return out
	}
	if newV == nil {
		out = append(out, ConfigChange{Path: path, OldValue: oldV, NewValue: nil})
		return out
	}

	ro, no := reflect.ValueOf(oldV), reflect.ValueOf(newV)
	ko, kn := ro.Kind(), no.Kind()

	// different kinds -> replace
	if ko != kn {
		return []ConfigChange{{Path: path, OldValue: oldV, NewValue: newV}}
	}

	switch ko {
	case reflect.Map:
		// both map[string]interface{}
		om := oldV.(map[string]interface{})
		nm := newV.(map[string]interface{})

		// check keys in old
		for k, ov := range om {
			newPath := join(path, k)
			if nv, ok := nm[k]; ok {
				out = append(out, diffValues(newPath, ov, nv)...)
			} else {
				out = append(out, ConfigChange{Path: newPath, OldValue: ov, NewValue: nil})
			}
		}
		// new-only keys
		for k, nv := range nm {
			if _, ok := om[k]; !ok {
				newPath := join(path, k)
				out = append(out, ConfigChange{Path: newPath, OldValue: nil, NewValue: nv})
			}
		}

	case reflect.Slice, reflect.Array:
		os := ro.Len()
		ns := no.Len()
		max := os
		if ns > max {
			max = ns
		}
		for i := 0; i < max; i++ {
			newPath := fmt.Sprintf("%s[%d]", path, i)
			var ov, nv interface{}
			if i < os {
				ov = ro.Index(i).Interface()
			}
			if i < ns {
				nv = no.Index(i).Interface()
			}
			out = append(out, diffValues(newPath, ov, nv)...)
		}

	default:
		// primitive or struct: DeepEqual
		if !reflect.DeepEqual(oldV, newV) {
			out = append(out, ConfigChange{Path: path, OldValue: oldV, NewValue: newV})
		}
	}

	return out
}

// join concatenates path and field
func join(base, field string) string {
	if base == "" {
		return field
	}
	return base + "." + field
}

// logConfigChanges logs the configuration changes
func logConfigChanges(changes []ConfigChange, logger iface.Logger) {
	if len(changes) == 0 {
		logger.Info("No changes detected in configuration.")
		return
	}

	// Group changes by section
	sections := make(map[string][]ConfigChange)
	for _, change := range changes {
		section := strings.Split(change.Path, ".")[0]
		sections[section] = append(sections[section], change)
	}

	// Create a title caser
	titleCaser := cases.Title(language.English)

	// Log changes by section
	for section, sectionChanges := range sections {
		logger.Info("%s changes:", titleCaser.String(section))
		for _, change := range sectionChanges {
			formatAndLogChange(change, logger)
		}
	}
}

// formatAndLogChange formats and logs a single change
func formatAndLogChange(change ConfigChange, logger iface.Logger) {
	// Additions
	if change.OldValue == nil && change.NewValue != nil {
		logger.Info("  - %s added (value: %v)", change.Path, change.NewValue)
		return
	}
	// Removals
	if change.NewValue == nil && change.OldValue != nil {
		logger.Info("  - %s removed (was: %v)", change.Path, change.OldValue)
		return
	}
	// Updates (both non-nil)
	switch oldVal := change.OldValue.(type) {
	case string:
		if newVal, ok := change.NewValue.(string); ok {
			logger.Info("  - %s changed from '%s' to '%s'", change.Path, oldVal, newVal)
		} else {
			logger.Info("  - %s changed from '%v' to '%v'", change.Path, change.OldValue, change.NewValue)
		}
	case bool:
		if newVal, ok := change.NewValue.(bool); ok {
			logger.Info("  - %s changed from %v to %v", change.Path, oldVal, newVal)
		} else {
			logger.Info("  - %s changed from %v to %v", change.Path, change.OldValue, change.NewValue)
		}
	case int, int8, int16, int32, int64, float32, float64:
		logger.Info("  - %s changed from %v to %v", change.Path, change.OldValue, change.NewValue)
	default:
		// Fallback for slices, maps, structs, etc.
		logger.Info("  - %s changed", change.Path)
	}
}

// sendConfigChangeTelemetry sends telemetry data for config changes
func sendConfigChangeTelemetry(ctx context.Context, changes []ConfigChange, logger iface.Logger) {
	if len(changes) == 0 {
		return
	}

	// Get metrics context
	metrics, err := telemetry.MetricsFromContext(ctx)
	if err != nil {
		logger.Warn("Error while getting telemetry client from context.", zap.Error(err))
	}

	// Add section change counts
	sectionCounts := make(map[string]int)
	for _, change := range changes {
		section := strings.Split(change.Path, ".")[0]
		sectionCounts[section]++
	}

	// Add individual changes (up to a reasonable limit)
	maxChangesToInclude := 20 // Avoid sending too much data
	changeDimensions := make(map[string]string)
	for i, change := range changes {
		if i >= maxChangesToInclude {
			logger.Warn("Reached max change limit of ", maxChangesToInclude, " for ", change.Path)
			break
		}

		fieldPath := fmt.Sprintf("changed_%d_path", i)
		changeDimensions[fieldPath] = change.Path

		// Only include primitive values that can be reasonably serialized
		oldValueStr := fmt.Sprintf("%v", change.OldValue)
		newValueStr := fmt.Sprintf("%v", change.NewValue)

		// Truncate long values
		const maxValueLen = 50
		if len(oldValueStr) > maxValueLen {
			oldValueStr = oldValueStr[:maxValueLen] + "..."
		}
		if len(newValueStr) > maxValueLen {
			newValueStr = newValueStr[:maxValueLen] + "..."
		}

		changeDimensions[fmt.Sprintf("changed_%d_from", i)] = oldValueStr
		changeDimensions[fmt.Sprintf("changed_%d_to", i)] = newValueStr
	}

	// Add section counts as properties
	for section, count := range sectionCounts {
		changeDimensions[section+"_changes"] = fmt.Sprintf("%d", count)
	}

	// Add change count as a metric
	metrics.AddMetricWithDimensions("ConfigChangeCount", float64(len(changes)), changeDimensions)
}
