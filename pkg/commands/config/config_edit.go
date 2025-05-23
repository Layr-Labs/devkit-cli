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

	"sigs.k8s.io/yaml"

	"github.com/urfave/cli/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type EditTarget int

const (
	Config EditTarget = iota
	Context
)

// editConfig is the main entry point for the edit config functionality
func EditConfig(cCtx *cli.Context, configPath string, editTarget EditTarget, context string) error {
	logger := common.LoggerFromContext(cCtx.Context)

	// Find an available editor
	editor, err := findEditor()
	if err != nil {
		return err
	}

	// Create a backup of the current config
	originalConfig, backupData, err := backupConfig(configPath, editTarget, context)
	if err != nil {
		return err
	}

	// Open the editor and wait for it to close
	if err := openEditor(editor, configPath, logger); err != nil {
		return err
	}

	// Validate the edited config
	newConfig, err := ValidateConfig(configPath)
	if err != nil {
		logger.Error("Error validating config: %v", err)
		logger.Info("Reverting changes...")
		if restoreErr := restoreBackup(configPath, backupData); restoreErr != nil {
			return fmt.Errorf("failed to restore backup after validation error: %w", restoreErr)
		}
		return err
	}

	// Collect changes
	changes := collectConfigChanges(originalConfig, newConfig, editTarget, logger)

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

var DefaultConfigPath = filepath.Join("config")

// backupConfig creates a backup of the current config
func backupConfig(configPath string, editTarget EditTarget, context string) (interface{}, []byte, error) {
	// Load the current config to compare later
	var (
		currentConfig interface{}
		err           error
	)

	// select the interface based on target
	switch editTarget {
	case Config:
		currentConfig, err = common.LoadBaseConfig()
	case Context:
		currentConfig, err = common.LoadContextConfig(context)
	}

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

// ValidateConfig checks if the edited config file is valid
func ValidateConfig(configPath string) (interface{}, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Try unmarshalling as BaseConfig (config.yaml)
	var base common.ConfigWithContextConfig
	if err := yaml.Unmarshal(data, &base); err == nil && base.Config.Project.Name != "" {
		return &base, nil
	}

	// Try unmarshalling as ChainContextConfig (devnet.yaml, sepolia.yaml)
	var ctxWrapper struct {
		Version string                    `yaml:"version"`
		Context common.ChainContextConfig `yaml:"context"`
	}
	if err := yaml.Unmarshal(data, &ctxWrapper); err == nil && ctxWrapper.Context.Name != "" {
		return &ctxWrapper, nil
	}

	return nil, fmt.Errorf("invalid YAML config: unrecognized structure")
}

// restoreBackup restores the original file content
func restoreBackup(configPath string, backupData []byte) error {
	return os.WriteFile(configPath, backupData, 0644)
}

// ConfigChange represents a change in a configuration field
type ConfigChange struct {
	Path     string
	OldValue interface{}
	NewValue interface{}
}

// collectConfigChanges compares the original and updated configurations
// and returns a slice of ConfigChange entries for all detected differences.
// It handles two edit targets: Config (base project config) and Context (chain context config).
func collectConfigChanges(original, updated interface{}, editTarget EditTarget, logger iface.Logger) []ConfigChange {
	var changes []ConfigChange

	switch editTarget {
	case Config:
		// Assert both original and updated to *ConfigWithContextConfig
		oldCfg, ok1 := original.(*common.ConfigWithContextConfig)
		newCfg, ok2 := updated.(*common.ConfigWithContextConfig)
		if !ok1 || !ok2 {
			// Log type mismatch and abort diff
			logger.Info("Mismatched types for Config comparison: %T vs %T", original, updated)
			return nil
		}
		// Compare only the Project block fields
		changes = getFieldChangesDetailed(
			"project",
			oldCfg.Config.Project,
			newCfg.Config.Project,
		)

	case Context:
		// Extract original ChainContextConfig pointer
		oldPtr, ok := original.(*common.ChainContextConfig)
		if !ok {
			logger.Info("Mismatched types for context.yaml comparison: %T", original)
			return nil
		}

		// Resolve updated ChainContextConfig pointer
		var newPtr *common.ChainContextConfig

		// Updated is directly *common.ChainContextConfig
		if nc, ok := updated.(*common.ChainContextConfig); ok {
			newPtr = nc
		} else {
			// Updated is an anonymous wrapper struct with a 'Context' field
			rv := reflect.ValueOf(updated)
			if rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.Struct {
				// Look for a field named 'Context'
				fv := rv.Elem().FieldByName("Context")
				if fv.IsValid() && fv.Type() == reflect.TypeOf(common.ChainContextConfig{}) {
					// Extract the inner ChainContextConfig value
					temp := fv.Interface().(common.ChainContextConfig)
					newPtr = &temp
				}
			}
		}
		if newPtr == nil {
			logger.Info("Mismatched updated type for Context: %T", updated)
			return nil
		}

		// Compare the flat fields on the ChainContextConfig
		changes = getFieldChangesDetailed(
			"context",
			*oldPtr,
			*newPtr,
		)

	default:
		// Unsupported edit target
		logger.Info("Unsupported edit target: %v", editTarget)
	}

	return changes
}

// getFieldChangesDetailed returns detailed field changes between two structs
func getFieldChangesDetailed(prefix string, old, new interface{}) []ConfigChange {
	changes := []ConfigChange{}

	// Use reflection to compare struct fields
	oldVal := reflect.ValueOf(old)
	newVal := reflect.ValueOf(new)

	// Handle nil values
	if !oldVal.IsValid() || !newVal.IsValid() {
		return changes
	}

	// Handle different types
	if oldVal.Type() != newVal.Type() {
		return changes
	}

	// Only handle struct types
	if oldVal.Kind() != reflect.Struct {
		return changes
	}

	// Compare all fields
	for i := 0; i < oldVal.NumField(); i++ {
		fieldName := oldVal.Type().Field(i).Name
		tomlTag := strings.Split(oldVal.Type().Field(i).Tag.Get("toml"), ",")[0]
		if tomlTag == "" {
			tomlTag = strings.ToLower(fieldName)
		}

		oldField := oldVal.Field(i)
		newField := newVal.Field(i)

		// Skip unexported fields
		if !oldField.CanInterface() {
			continue
		}

		// Skip complex nested structures (they'll be handled separately)
		if oldField.Kind() == reflect.Struct || oldField.Kind() == reflect.Map ||
			(oldField.Kind() == reflect.Slice && oldField.Type().Elem().Kind() != reflect.String) {
			continue
		}

		// Compare values
		if !reflect.DeepEqual(oldField.Interface(), newField.Interface()) {
			fieldPath := fmt.Sprintf("%s.%s", prefix, tomlTag)
			changes = append(changes, ConfigChange{
				Path:     fieldPath,
				OldValue: oldField.Interface(),
				NewValue: newField.Interface(),
			})
		}
	}

	return changes
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
	var changeMsg string

	// Format based on change type
	switch oldVal := change.OldValue.(type) {
	case string:
		if newVal, ok := change.NewValue.(string); ok && newVal != "removed" && newVal != "added" {
			changeMsg = fmt.Sprintf("%s changed from '%v' to '%v'", change.Path, oldVal, newVal)
		} else if newVal == "removed" {
			changeMsg = fmt.Sprintf("%s removed", change.Path)
		} else {
			changeMsg = fmt.Sprintf("%s changed", change.Path)
		}
	case bool:
		if newVal, ok := change.NewValue.(bool); ok {
			changeMsg = fmt.Sprintf("%s changed from %v to %v", change.Path, oldVal, newVal)
		} else {
			changeMsg = fmt.Sprintf("%s changed", change.Path)
		}
	case int, int8, int16, int32, int64:
		changeMsg = fmt.Sprintf("%s changed from %v to %v", change.Path, oldVal, change.NewValue)
	default:
		if change.NewValue == "added" {
			changeMsg = fmt.Sprintf("%s added", change.Path)
		} else if change.NewValue == "removed" {
			changeMsg = fmt.Sprintf("%s removed", change.Path)
		} else if change.NewValue == "modified" {
			changeMsg = fmt.Sprintf("%s modified", change.Path)
		} else {
			changeMsg = fmt.Sprintf("%s changed", change.Path)
		}
	}

	logger.Info("  - %s", changeMsg)
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
