# EigenLayer Development Kit

A CLI tool for developing and managing EigenLayer AVS (Autonomous Verifiable Services) projects.

## Quick Start

```bash
# Clone and build
git clone <repository-url>
cd devkit

# Build using make
make install

# Or build manually
go build -o devkit ./cmd/devkit

# Get started
devkit --help
```

## Development

```bash
make help      # Show all commands
make build     # Build binary
make tests     # Run tests
make lint      # Run linter

# Install pre-commit hooks
pre-commit install
```

## Core Commands

- `devkit avs create` - Scaffold new AVS projects
- `devkit avs config` - Manage project configuration
- `devkit avs build` - Compile contracts and binaries
- `devkit avs devnet` - Run local development network
- `devkit avs run` - Execute and simulate tasks
- `devkit avs release` - Package for deployment

## Options

- `--verbose, -v` - Enable detailed logging
- `--help, -h` - Show command help

## Example

```bash
devkit avs create MyAVS --lang go
devkit avs devnet start --fork base
```

## Telemetry

The CLI collects anonymous usage data to help improve the tool. This includes:
- Command usage (which commands are run)
- Basic system information (OS, architecture)
- Command execution time
- Errors encountered

No personal information or project details are collected. You can disable telemetry:
- Use the `--no-telemetry` flag when running create command

## For Developers

Adding custom telemetry metrics is simple with a single line of code: 
Example in a command implementation:

```go
Action: func(cCtx *cli.Context) error {
    // ... 
    // Track a custom event with properties
    props := map[string]interface{}{
        "port": cCtx.Int("port"),
        "contract_count": 5,
    }
    hooks.Track(cCtx.Context, hooks.FormatEventName("avs_devnet", "containers_up"), props)
    
    return nil
}
```

Standard metrics like command invocation, completion, and errors are tracked automatically.
