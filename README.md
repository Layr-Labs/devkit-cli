# EigenLayer Development Kit (DevKit) 🚀

**A CLI toolkit for developing, testing, and managing EigenLayer Autonomous Verifiable Services (AVS).**

EigenLayer DevKit streamlines AVS development, enabling you to quickly scaffold projects, compile contracts, run local networks, and simulate tasks with ease.

*(Graphic placeholder: Insert visual flow diagram here.)*

---

## 🌟 Key Commands Overview

| Command      | Description                              |
| ------------ | ---------------------------------------- |
| `avs create` | Scaffold a new AVS project               |
| `avs config` | Configure your AVS (`eigen.toml`)        |
| `avs build`  | Compile AVS smart contracts and binaries |
| `avs devnet` | Manage local development network         |
| `avs run`    | Simulate AVS task execution locally      |

---

## 🚦 Getting Started

### ✅ Prerequisites

Before you begin, ensure you have:

* [Docker](https://docs.docker.com/engine/install/)
* [Go](https://go.dev/doc/install)
* [Foundry](https://book.getfoundry.sh/getting-started/installation)
* [yq](https://github.com/mikefarah/yq/#install)

### 📦 Installation

Clone and build the DevKit CLI:

```bash
git clone https://github.com/Layr-Labs/devkit-cli
cd devkit-cli
go build -o devkit ./cmd/devkit
export PATH=$PATH:~/bin
```

Verify your installation:

```bash
devkit --help
```

### 🔑 Setup for Private Go Modules

During this Private Preview (closed beta), you'll need access to private Go modules hosted on GitHub:

1. **Add SSH Key to GitHub:** Ensure your SSH key is associated with your GitHub account ([instructions](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/adding-a-new-ssh-key-to-your-github-account)).
2. **Verify Repository Access:** Confirm with EigenLabs support that your account has access to necessary private repositories.
3. **Configure Git for SSH Access:**

```bash
git config --global url."ssh://git@github.com/Layr-Labs/".insteadOf "https://github.com/Layr-Labs/"
```

If you're on MacOS, ensure your `~/.ssh/config` does not contain `UseKeychain yes`, as it can interfere with SSH operations.

---

## 🚧 Step-by-Step Guide

### 1️⃣ Create a New AVS Project

Quickly scaffold your new AVS project:

* Initializes a new project based on the default task-based architecture in Go.
* Generates boilerplate code and default configuration.

Projects are created by default in `/Users/[current-user]/avs/`:

```bash
devkit avs create my-avs-project
cd /Users/[current-user]/avs/my-avs-project
```

> \[!IMPORTANT]
> All subsequent `devkit avs` commands must be run from the root of your AVS project—the directory containing the `eigen.toml` file. If `eigen.toml` is missing or located elsewhere, the CLI will fail to load the configuration.

### 2️⃣ Configure Your AVS (`eigen.toml`)

Customize project settings to define operators, network configurations, and more. You can configure this file either through the CLI or by manually editing the `eigen.toml` file.

View current settings via CLI:

```bash
devkit avs config
```

Edit settings directly via CLI:

```bash
devkit avs config --edit
```

Alternatively, manually edit `eigen.toml` in a text editor of your choice.

> \[!IMPORTANT]
> These commands must be run from your AVS project's root directory.

### 3️⃣ Build Your AVS

Compile AVS smart contracts and binaries to prepare your service for local execution:

* Compiles smart contracts using Foundry.
* Builds operator, aggregator, and AVS logic binaries.

Ensure you're in your project directory before running:

```bash
devkit avs build
```

### 4️⃣ Launch Local DevNet

Start a local Ethereum-based development network to simulate your AVS environment:

* Uses `eigenlayer-contracts-1.3.0` on a fresh Anvil chain.
* Automatically funds wallets (`operator_keys` and `submit_wallet`) if balances are below `10 ether`.
* Deploys required AVS and EigenLayer core contracts.
* Initializes aggregator and executor processes.

> \[!IMPORTANT]
> Please ensure your Docker daemon is running beforehand.

Run this from your project directory:

```bash
devkit avs devnet start
```

DevNet management commands:

| Command | Description                                 |
| ------- | ------------------------------------------- |
| `start` | Start local Docker containers and contracts |
| `stop`  | Stop and remove containers and resources    |
| `list`  | List active containers and their ports      |

### 5️⃣ Simulate Task Execution (`avs run`)

Test your AVS logic locally by simulating task execution:

* Simulate the full lifecycle of task submission and execution.
* Validate both off-chain and on-chain logic.
* Review detailed execution results.

Run this from your project directory:

```bash
devkit avs run
```

Optionally, submit tasks directly to the on-chain TaskMailBox contract via a frontend or another method for more realistic testing scenarios.

---

## 📖 Logging and Telemetry

Configure logging levels through `eigen.toml`:

```toml
[log]
level = "info"  # Options: "info", "debug", "warn", "error"
```

To enable detailed logging during commands:

```bash
devkit --verbose avs build
```

---

## 🌍 Environment Variables

DevKit automatically loads environment variables from a `.env` file in your project directory:

```bash
cp .env.example .env
nano .env
```

---

## 🤝 Contributing

Contributions are welcome! Please open an issue to discuss significant changes before submitting a pull request.
