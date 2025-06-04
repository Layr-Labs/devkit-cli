package common

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
)

// TelemetryPrompt presents the telemetry opt-in dialog to first-time users
func TelemetryPrompt(logger iface.Logger) (bool, error) {
	// Display telemetry information
	fmt.Println()
	fmt.Println("🎯 Welcome to EigenLayer DevKit!")
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📊 Help us improve DevKit by sharing anonymous usage data")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("We'd like to collect anonymous usage data to help us improve DevKit.")
	fmt.Println()
	fmt.Println("This includes:")
	fmt.Println("  • Commands used (e.g., 'devkit avs create', 'devkit avs build')")
	fmt.Println("  • Error counts and types (to identify common issues)")
	fmt.Println("  • Country and city details (to help us understand global usage)")
	fmt.Println("  • Performance metrics (command execution times)")
	fmt.Println("  • System information (OS, architecture)")
	fmt.Println()
	fmt.Println("We do NOT collect:")
	fmt.Println("  • Personal information")
	fmt.Println("  • Private keys or sensitive data")
	fmt.Println()
	fmt.Println("You can change this setting anytime with:")
	fmt.Println("  devkit telemetry --enable   # Enable telemetry")
	fmt.Println("  devkit telemetry --disable  # Disable telemetry")
	fmt.Println()
	fmt.Print("Would you like to enable telemetry? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))

	// Default to yes if empty response, no if they explicitly say no
	enabled := response == "" || response == "y" || response == "Y" || response == "yes" || response == "Yes"

	if enabled {
		fmt.Println("✅ Telemetry enabled. Thank you for helping improve DevKit!")
	} else {
		fmt.Println("❌ Telemetry disabled. You can enable it later if you change your mind.")
	}
	fmt.Println()

	return enabled, nil
}

// HandleFirstRunTelemetryPrompt checks if this is a first run and prompts for telemetry
// Returns the telemetry preference (true/false) and whether this was a first run
func HandleFirstRunTelemetryPrompt(logger iface.Logger) (bool, bool, error) {
	// Check if this is the first run
	isFirstRun, err := IsFirstRun()
	if err != nil {
		logger.Debug("Failed to check first run status: %v", err)
		// Don't fail the entire command, just assume not first run
		return false, false, nil
	}

	// If not first run, get existing global preference
	if !isFirstRun {
		preference, err := GetGlobalTelemetryPreference()
		if err != nil {
			logger.Debug("Failed to get global telemetry preference: %v", err)
			return false, false, nil
		}

		if preference != nil {
			return *preference, false, nil
		}

		// No preference set, default to false
		return false, false, nil
	}

	// Telemetry is configurable, show the prompt
	telemetryEnabled, err := TelemetryPrompt(logger)
	if err != nil {
		logger.Debug("Failed to show telemetry prompt: %v", err)
		// Default to disabled if prompt fails
		telemetryEnabled = false
	}

	// Save the preference globally
	if err := SetGlobalTelemetryPreference(telemetryEnabled); err != nil {
		logger.Debug("Failed to save global telemetry preference: %v", err)
	}

	// Mark first run as complete and save version info
	if err := markFirstRunCompleteWithVersion(); err != nil {
		logger.Debug("Failed to mark first run complete: %v", err)
	}

	return telemetryEnabled, true, nil
}

// markFirstRunCompleteWithVersion marks first run complete and records version
func markFirstRunCompleteWithVersion() error {
	config, err := LoadGlobalConfig()
	if err != nil {
		return err
	}

	config.FirstRun = false

	return SaveGlobalConfig(config)
}
