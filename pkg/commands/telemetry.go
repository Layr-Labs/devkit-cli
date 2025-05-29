package commands

import (
	"fmt"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/urfave/cli/v2"
)

// TelemetryCommand allows users to manage telemetry settings
var TelemetryCommand = &cli.Command{
	Name:  "telemetry",
	Usage: "Manage telemetry settings",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "enable",
			Usage: "Enable telemetry collection",
		},
		&cli.BoolFlag{
			Name:  "disable",
			Usage: "Disable telemetry collection",
		},
		&cli.BoolFlag{
			Name:  "status",
			Usage: "Show current telemetry status",
		},
		&cli.BoolFlag{
			Name:  "global",
			Usage: "Apply setting globally (affects all projects and global default)",
		},
	},
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx.Context)

		enable := cCtx.Bool("enable")
		disable := cCtx.Bool("disable")
		status := cCtx.Bool("status")
		global := cCtx.Bool("global")

		// Validate flags
		if (enable && disable) || (!enable && !disable && !status) {
			return fmt.Errorf("specify exactly one of --enable, --disable, or --status")
		}

		if status {
			return showTelemetryStatus(logger, global)
		}

		if enable {
			return enableTelemetry(logger, global)
		}

		if disable {
			return disableTelemetry(logger, global)
		}

		return nil
	},
}

// displayGlobalTelemetryStatus shows the global telemetry preference status
func displayGlobalTelemetryStatus(logger iface.Logger, prefix string) error {
	globalPreference, err := common.GetGlobalTelemetryPreference()
	if err != nil {
		return fmt.Errorf("failed to get global telemetry preference: %w", err)
	}

	if globalPreference == nil {
		logger.Info("%s: Not set (defaults to disabled)", prefix)
	} else if *globalPreference {
		logger.Info("%s: Enabled", prefix)
	} else {
		logger.Info("%s: Disabled", prefix)
	}
	return nil
}

func showTelemetryStatus(logger iface.Logger, global bool) error {
	if global {
		return displayGlobalTelemetryStatus(logger, "Global telemetry")
	}

	// Show effective status (project takes precedence over global)
	effectivePreference, err := common.GetEffectiveTelemetryPreference()
	if err != nil {
		// If not in a project, show global preference
		return displayGlobalTelemetryStatus(logger, "Telemetry")
	}

	// Check if we're in a project and if there's a project-specific setting
	projectSettings, projectErr := common.LoadProjectSettings()
	if projectErr == nil && projectSettings != nil {
		if effectivePreference {
			logger.Info("Telemetry: Enabled (project setting)")
		} else {
			logger.Info("Telemetry: Disabled (project setting)")
		}

		// Also show global setting for context
		return displayGlobalTelemetryStatus(logger, "Global default")
	} else {
		// Not in project, show global
		if effectivePreference {
			logger.Info("Telemetry: Enabled (global setting)")
		} else {
			logger.Info("Telemetry: Disabled (global setting)")
		}
	}

	return nil
}

func enableTelemetry(logger iface.Logger, global bool) error {
	if global {
		// Set global preference only
		if err := common.SetGlobalTelemetryPreference(true); err != nil {
			return fmt.Errorf("failed to enable global telemetry: %w", err)
		}

		logger.Info("✅ Global telemetry enabled")
		logger.Info("New projects will inherit this setting.")
		return nil
	}

	// Set project-specific preference
	if err := common.SetProjectTelemetry(true); err != nil {
		return fmt.Errorf("failed to enable project telemetry: %w", err)
	}

	logger.Info("✅ Telemetry enabled for this project")
	return nil
}

func disableTelemetry(logger iface.Logger, global bool) error {
	if global {
		// Set global preference only
		if err := common.SetGlobalTelemetryPreference(false); err != nil {
			return fmt.Errorf("failed to disable global telemetry: %w", err)
		}

		logger.Info("❌ Global telemetry disabled")
		logger.Info("New projects will inherit this setting.")
		return nil
	}

	// Set project-specific preference
	if err := common.SetProjectTelemetry(false); err != nil {
		return fmt.Errorf("failed to disable project telemetry: %w", err)
	}

	logger.Info("❌ Telemetry disabled for this project")
	return nil
}
