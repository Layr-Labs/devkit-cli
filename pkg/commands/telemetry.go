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

func showTelemetryStatus(logger iface.Logger, global bool) error {
	if global {
		// Show global status
		globalPreference, err := common.GetGlobalTelemetryPreference()
		if err != nil {
			return fmt.Errorf("failed to get global telemetry preference: %w", err)
		}

		if globalPreference == nil {
			fmt.Println("Global telemetry: Not set (defaults to disabled)")
		} else if *globalPreference {
			fmt.Println("Global telemetry: Enabled")
		} else {
			fmt.Println("Global telemetry: Disabled")
		}
		return nil
	}

	// Show effective status (project takes precedence over global)
	effectivePreference, err := common.GetEffectiveTelemetryPreference()
	if err != nil {
		// If not in a project, show global preference
		globalPreference, globalErr := common.GetGlobalTelemetryPreference()
		if globalErr != nil {
			return fmt.Errorf("failed to get telemetry preferences: %w", globalErr)
		}

		if globalPreference == nil {
			fmt.Println("Telemetry: Disabled (no preference set)")
		} else if *globalPreference {
			fmt.Println("Telemetry: Enabled (global setting)")
		} else {
			fmt.Println("Telemetry: Disabled (global setting)")
		}
		return nil
	}

	// Check if we're in a project and if there's a project-specific setting
	projectSettings, projectErr := common.LoadProjectSettings()
	if projectErr == nil && projectSettings != nil {
		if effectivePreference {
			fmt.Println("Telemetry: Enabled (project setting)")
		} else {
			fmt.Println("Telemetry: Disabled (project setting)")
		}

		// Also show global setting for context
		globalPreference, _ := common.GetGlobalTelemetryPreference()
		if globalPreference == nil {
			fmt.Println("Global default: Not set")
		} else if *globalPreference {
			fmt.Println("Global default: Enabled")
		} else {
			fmt.Println("Global default: Disabled")
		}
	} else {
		// Not in project, show global
		if effectivePreference {
			fmt.Println("Telemetry: Enabled (global setting)")
		} else {
			fmt.Println("Telemetry: Disabled (global setting)")
		}
	}

	return nil
}

func enableTelemetry(logger iface.Logger, global bool) error {
	if global {
		// Set global preference
		if err := common.SetGlobalTelemetryPreference(true); err != nil {
			return fmt.Errorf("failed to enable global telemetry: %w", err)
		}

		// Also update current project if we're in one
		if err := common.SetProjectTelemetry(true); err != nil {
			// Don't fail if we're not in a project
			logger.Info("Global telemetry enabled. Note: Not in a project directory.")
		} else {
			logger.Info("Global telemetry enabled and applied to current project.")
		}

		fmt.Println("✅ Telemetry enabled globally")
		return nil
	}

	// Set project-specific preference
	if err := common.SetProjectTelemetry(true); err != nil {
		return fmt.Errorf("failed to enable project telemetry: %w", err)
	}

	fmt.Println("✅ Telemetry enabled for this project")
	return nil
}

func disableTelemetry(logger iface.Logger, global bool) error {
	if global {
		// Set global preference
		if err := common.SetGlobalTelemetryPreference(false); err != nil {
			return fmt.Errorf("failed to disable global telemetry: %w", err)
		}

		// Also update current project if we're in one
		if err := common.SetProjectTelemetry(false); err != nil {
			// Don't fail if we're not in a project
			logger.Info("Global telemetry disabled. Note: Not in a project directory.")
		} else {
			logger.Info("Global telemetry disabled and applied to current project.")
		}

		fmt.Println("❌ Telemetry disabled globally")
		return nil
	}

	// Set project-specific preference
	if err := common.SetProjectTelemetry(false); err != nil {
		return fmt.Errorf("failed to disable project telemetry: %w", err)
	}

	fmt.Println("❌ Telemetry disabled for this project")
	return nil
}
