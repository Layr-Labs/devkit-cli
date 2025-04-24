# EigenLayer Development Kit (DevKit)

A CLI tool for developing and managing EigenLayer AVS (Autonomous Verifiable Services) projects.

---

## 🚀 Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/engine/install/)
- [Go](https://go.dev/doc/install)
- [Foundry](https://book.getfoundry.sh/getting-started/installation)
- [make](https://formulae.brew.sh/formula/make)


```bash
# Clone the repository
git clone https://github.com/Layr-Labs/devkit-cli
cd devkit-cli

# Install the CLI
make install

# Or build manually
go build -o devkit ./cmd/devkit

# Get started
devkit --help
```

---

## 🛠️ Development Workflow

```bash
make help      # Show all available dev commands
make build     # Build CLI binary
make tests     # Run all unit tests
make lint      # Run linter and static checks
```

---


## 💻 Core DevKit Commands
> [!IMPORTANT]  
> All <code>devkit avs</code> commands must be run from the root of your AVS project — the directory that contains the <code>eigen.toml</code> file.  
> If <code>eigen.toml</code> is missing or located elsewhere, the CLI will fail to load the project configuration.

| Command                     | Description                                 |
|----------------------------|---------------------------------------------|
| `devkit avs create`        | Scaffold a new AVS project                  |
| `devkit avs config`        | Read or modify `eigen.toml` configuration   |
| `devkit avs build`         | Compile smart contracts and binaries        |
| `devkit avs devnet`        | Start/stop a local Docker-based devnet      |
| `devkit avs run`           | Simulate and execute AVS tasks locally      |
| `devkit avs release`       | Package your AVS for testnet/mainnet        |

### Devnet 
> [!Warning]
> Docker daemon must be running beforehand.
#### Starting the devnet 
```bash
devkit avs devnet start 
```
#### Stopping the devnet 
```bash
devkit avs devnet stop
```

## ⚙️ Global Options

| Flag             | Description            |
|------------------|------------------------|
| `--verbose`, `-v`| Enable verbose logging |
| `--help`, `-h`   | Show help output       |

---

## 💡 Example Usage
```bash
# Scaffold a new AVS named MyAVS
devkit avs create MyAVS --lang go

# Start a local devnet
devkit avs devnet start

# Check block number using cast
cast block-number --rpc-url http://localhost:8545

# Stop the devnet
devkit avs devnet stop
```

## 🤝 Contributing
Pull requests are welcome! For major changes, open an issue first to discuss what you would like to change.
