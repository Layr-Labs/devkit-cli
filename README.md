# EigenLayer Development Kit (DevKit) 🚀

**A CLI toolkit for developing, testing, and managing EigenLayer Autonomous Verifiable Services (AVS).**

EigenLayer DevKit streamlines AVS development, enabling you to quickly scaffold projects, compile contracts, run local networks, and simulate tasks with ease.

![EigenLayer DevKit User Flow](assets/devkit-user-flow.png)

---

## 🌟 Key Commands Overview

| Command      | Description                              |
| ------------ | ---------------------------------------- |
| `avs create` | Scaffold a new AVS project               |
| `avs config` | Configure your AVS (`config/config.yaml`,`config/devnet.yaml`...)        |
| `avs build`  | Compile AVS smart contracts and binaries |
| `avs devnet` | Manage local development network         |
| `avs run`    | Simulate AVS task execution locally      |

---

## 🚦 Getting Started

### ✅ Prerequisites

Before you begin, ensure you have:

* [Docker](https://docs.docker.com/engine/install/)
* [Go](https://go.dev/doc/install)
* [make](https://formulae.brew.sh/formula/make)
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

---

## 🚧 Step-by-Step Guide

### 1️⃣ Create a New AVS Project

Quickly scaffold your new AVS project:

* Initializes a new project based on the default task-based architecture in Go.
* Generates boilerplate code and default configuration.

Projects are created by default in the current directory from where the below command is called.

```bash
devkit avs create my-avs-project
cd my-avs-project
```

> \[!IMPORTANT]
> All subsequent `devkit avs` commands must be run from the root of your AVS project—the directory containing the [config](https://github.com/Layr-Labs/devkit-cli/tree/main/config) folder . The `config` folder contains the base `config.yaml` with the `contexts` folder which houses the respective context yaml files , example `devnet.yaml`.

### 2️⃣ Configure Your AVS (`config.yaml`,`devnet.yaml`)

Customize project settings to define operators, network configurations, and more. You can configure this file either through the CLI or by manually editing the `config.yaml` and `contexts/devnet.yaml` files.
View current settings via CLI:

```bash
devkit avs config
```

Edit settings directly via CLI:

```bash
devkit avs config --edit --path <path to the config.yaml or contexts/devnet.yaml file>
```

Alternatively, manually edit `` in a text editor of your choice.

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

* Forks ethereum mainnet using a fork url which the user passes along with the block number.
* Automatically funds wallets (`operator_keys` and `submit_wallet`) if balances are below `10 ether`.
* Setup required AVS contracts.
* Initializes aggregator and executor processes.

> \[!IMPORTANT]
> Please ensure your Docker daemon is running beforehand.

Run this from your project directory:

```bash
devkit avs devnet start
```

DevNet management commands:

| Command | Description                                                             |
| ------- | -------------------------------------------                             |
| `start` | Start local Docker containers and contracts                             |
| `stop`  | Stop and remove container from the avs project this command is called   |
| `list`  | List active containers and their ports                                  |
| `stop --all`  | Stops all devkit devnet containers that are currently currening                                  |
| `stop --project.name`  | Stops the specific project's devnet                                  |
| `stop --port`  | Stops the specific port .ex: `stop --port 8545`                                  |


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

## Bls keystore 
You can create and read keystore for bn254 private keys using our cli 

- To create a keystore from a given private key in BigInt
```bash
devkit keystore create --key <private key in big int> --path <path to json file(It needs to include the filename ex: ./keystores/operator1.keystore.json)> --password <password for the keystore>
```

- To read an existing keystore json using the password
```bash
devkit keystore read --path <./keystores/operator1.keystore.json> --password <password>
```

## 📖 Logging

<!-- 
@TODO: bring this back when we reintroduce config log levels
Configure logging levels through `config.yaml`:

```yaml
log:
  level: info  # Options: "info", "debug", "warn", "error"
``` -->

To enable detailed logging during commands:

```bash
devkit avs build --verbose
```

---

## 🌍 Environment Variables

The DevKit CLI automatically loads environment variables from a `.env` file in your project directory:

- If a `.env` file exists in your project directory, its variables will be loaded for all commands except `create`
- Template repositories should include a `.env.example` file that you can copy to `.env` and modify
- This is useful for storing configuration that shouldn't be committed to version control (API keys, private endpoints, etc.)

Example workflow:
```bash
# After creating a project from a template
cd my-avs-project

# Copy the example env file (if provided by the template)
cp .env.example .env

# Edit with your specific values
nano .env

# Run commands - the .env file will be automatically loaded
devkit avs build
devkit avs run
```

---

## 🤝 Contributing

Contributions are welcome! Please open an issue to discuss significant changes before submitting a pull request.
