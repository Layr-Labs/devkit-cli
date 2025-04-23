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
