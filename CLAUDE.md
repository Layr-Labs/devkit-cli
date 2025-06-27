# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

EigenLayer DevKit is a CLI toolkit for scaffolding, developing, and testing EigenLayer Autonomous Verifiable Services (AVS). It's built in Go and focuses on the Hourglass task-based architecture. The DevKit is currently in closed alpha and intended for local experimentation and development only.

## Common Development Commands

### Building and Testing
```bash
# Build the CLI binary
make build

# Run all tests (may be slow)
make tests

# Run fast tests (skips slow integration tests) 
make tests-fast

# Install binary to ~/bin/ and set up shell completion
make install

# Format code
make fmt

# Run linter
make lint

# Clean up build artifacts
make clean
```

### Testing the CLI
After building, test the CLI:
```bash
./bin/devkit --help
./bin/devkit avs --help
```

### Cross-platform Builds
```bash
# Build for specific platforms
make build/darwin-arm64
make build/darwin-amd64  
make build/linux-arm64
make build/linux-amd64

# Build all platforms
make release
```

## Architecture Overview

### CLI Command Structure
The CLI is built with `urfave/cli/v2` and organized hierarchically:
- **Main entry**: `cmd/devkit/main.go`
- **Core commands**: All under `devkit avs` subcommand
- **Command implementations**: `pkg/commands/` directory

Key commands:
- `devkit avs create` - Scaffold new AVS projects from templates
- `devkit avs build` - Compile contracts and binaries via template scripts
- `devkit avs devnet` - Manage local Docker-based development networks
- `devkit avs call` - Simulate task execution
- `devkit avs config/context` - Configuration management

### Configuration System
Multi-layered configuration with migration support:

1. **Global Config** (`~/.config/devkit/config.yaml`): User preferences, telemetry settings
2. **Project Config** (`config/config.yaml`): Project metadata, template info
3. **Context Config** (`config/contexts/{context}.yaml`): Environment-specific settings (devnet, testnet, mainnet)

**Current Versions**: Config v0.0.2, Context v0.0.6

The system includes automatic migrations between versions via `pkg/migration/` that preserve user customizations.

### Template System Architecture
Projects are scaffolded from versioned Git templates:
- **Template registry**: `config/templates.yaml` defines available templates
- **Template fetching**: `pkg/template/git_fetcher.go` handles Git operations
- **Project initialization**: Templates provide `.devkit/scripts/init` for setup
- **Build/run integration**: Templates provide `.devkit/scripts/build` and `.devkit/scripts/run`

### Devnet System
The devnet management system (`pkg/commands/devnet.go`) provides:
- Local Docker-based Anvil chains with EigenLayer state forked from Holesky
- Automated contract deployment (L1/L2)
- Pre-funded test operators with BLS keystores
- AVS registration and operator management

### Package Organization
- **`pkg/commands/`**: CLI command implementations
- **`pkg/common/`**: Shared utilities, configuration, contracts, logging
- **`pkg/template/`**: Git-based template management
- **`pkg/telemetry/`**: PostHog analytics integration  
- **`pkg/migration/`**: Configuration migration system
- **`pkg/hooks/`**: Command lifecycle hooks

## Key Dependencies

- **Go 1.23.6+** required
- **EigenLayer contracts**: `github.com/Layr-Labs/eigenlayer-contracts`
- **Hourglass AVS**: `github.com/Layr-Labs/hourglass-monorepo/ponos`
- **External tools**: Docker, Foundry, Zeus (npm package `@layr-labs/zeus@1.5.2`)

## Development Environment Setup

1. Install prerequisites: Docker, Foundry, Go 1.23.6+, make, jq, yq
2. Clone repository and run `make install`
3. Zeus is automatically installed as npm global package during `make install`

## Testing Patterns

- Unit tests use standard Go testing
- Integration tests may require Docker and external dependencies
- Use `make tests-fast` for quick feedback during development
- Integration tests in `test/integration/` directory

## Configuration Migration

When adding new configuration fields:
1. Update config structs in `pkg/common/`
2. Create migration in `config/configs/migrations/` or `config/contexts/migrations/`
3. Update embedded config versions in `config/`
4. Test migration with existing project configs

## Template Development

Templates must provide:
- `.devkit/scripts/init` - Project initialization
- `.devkit/scripts/build` - Build script for contracts/binaries
- `.devkit/scripts/run` - Run script for AVS components
- Standard Go project structure for task-based architecture

## Telemetry System

Optional PostHog-based telemetry with:
- Global and project-level opt-in/opt-out
- Privacy-conscious data collection
- CI environment auto-detection (defaults to disabled)