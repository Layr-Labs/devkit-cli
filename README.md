# EigenLayer Development Kit

A CLI tool for developing and managing EigenLayer AVS (Actively Validated Services) projects.

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

## Core Commands (under devkit)

- `avs create` - Scaffold new AVS projects
- `avs config` - Manage project configuration
- `avs build` - Compile contracts and binaries
- `avs devnet` - Run local development network
- `avs run` - Execute and simulate tasks
- `avs release` - Package for deployment

## Development

```bash
make help      # Show all commands
make build     # Build binary
make tests     # Run tests
make lint      # Run linter
```

## Options

- `--verbose, -v` - Enable detailed logging
- `--help, -h` - Show command help

## Example

```bash
devkit create MyAVS --lang go
devkit devnet start --fork base
```
