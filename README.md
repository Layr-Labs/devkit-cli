## ⚠️ Disclaimer: Closed Alpha Not Production Ready
EigenLayer DevKit is currently in a closed alpha stage and is intended strictly for local experimentation and development. It has not been audited, and should not be used in any live environment, including public testnets or mainnet. Users are strongly discouraged from pushing generated projects to remote repositories without reviewing and sanitizing sensitive configuration files (e.g. devnet.yaml), which may contain private keys or other sensitive material.

# EigenLayer Development Kit (DevKit) 🚀

**A CLI toolkit for scaffolding, developing, and testing EigenLayer Autonomous Verifiable Services (AVS).**

EigenLayer DevKit streamlines AVS development, enabling you to:
* Quickly scaffold projects
* Compile contracts
* Run local networks
* Simulate tasks

Use DevKit to get from AVS idea to Proof of Concept with a local testing environment that includes task simulation.

> **Note:** The current DevKit features support local experimentation, development, and testing of AVS using the Hourglass task-based framework. We're actively expanding capabilities, so if there's a gap for your scenario, check out our roadmap to see what's coming, or let us know what would support you in building AVS.

![EigenLayer DevKit User Flow](assets/devkit-user-flow.png)

## 🌟 Key Commands Overview

| Command        | Description                                                       |
|----------------|-------------------------------------------------------------------|
| `devkit avs create`   | Scaffold a new AVS project                                        |
| `devkit avs config`   | Configure your Project (`config/config.yaml`) |
| `devkit avs context`   | Configure your environment and AVS (`config/devnet.yaml`...) |
| `devkit avs build`    | Compile AVS smart contracts and binaries                          |
| `devkit avs devnet`   | Manage local development network                                  |
| `devkit avs call`     | Simulate AVS task execution locally                               |


---

## 🚦 Getting Started

### ✅ Prerequisites

* [Docker (latest)](https://docs.docker.com/engine/install/)
* [Foundry (latest)](https://book.getfoundry.sh/getting-started/installation)
* [Go (v1.23.6)](https://go.dev/doc/install)
* [Gomplate (v4)](https://docs.gomplate.ca/installing/)
* [make (v4.3)](https://formulae.brew.sh/formula/make)
* [jq (v1.7.1)](https://jqlang.org/download/)
* [yq (v4.35.1)](https://github.com/mikefarah/yq/#install)
* [zeus (v1.5.2)](https://github.com/Layr-Labs/zeus)

On macOS and Debian, running the following command installs all required dependencies and version numbers automatically. For other OSs, manual installation of software prerequisites is required:

```bash
devkit avs create my-avs-project ./
```



### 📦 Installation

To download a binary for the latest release, run:
```bash
# macOS (Apple Silicon)
mkdir -p $HOME/bin && curl -sL https://s3.amazonaws.com/eigenlayer-devkit-releases/v0.0.9/devkit-darwin-arm64-v0.0.9.tar.gz | tar xv -C "$HOME/bin"

# macOS (Intel)
mkdir -p $HOME/bin && curl -sL https://s3.amazonaws.com/eigenlayer-devkit-releases/v0.0.9/devkit-darwin-amd64-v0.0.9.tar.gz | tar xv -C "$HOME/bin"

# Linux (x86_64 / AMD64)
mkdir -p $HOME/bin && curl -sL https://s3.amazonaws.com/eigenlayer-devkit-releases/v0.0.9/devkit-linux-amd64-v0.0.9.tar.gz | tar xv -C "$HOME/bin"

# Linux (ARM64 / aarch64)
mkdir -p $HOME/bin && curl -sL https://s3.amazonaws.com/eigenlayer-devkit-releases/v0.0.9/devkit-linux-arm64-v0.0.9.tar.gz | tar xv -C "$HOME/bin"
```

The binary will be installed inside the ~/bin directory.

To add the binary to your path, run:
```bash
export PATH=$PATH:~/bin
```

To build and install the devkit cli from source:
```bash
mkdir -p $HOME/bin
git clone https://github.com/Layr-Labs/devkit-cli
cd devkit-cli
make install
export PATH=$PATH:~/bin
```

Verify your installation:
```bash
devkit --help
```

### 🔧 Shell Completion (Optional)

Tab completion for devkit commands is automatically set up when you install with `make install`.

**If you installed from source with `make install`:**
- Completion is automatically configured and enabled! Test it immediately:
```bash
devkit <TAB>          # Should show: avs, keystore, version
devkit avs <TAB>      # Should show subcommands
```

**If you downloaded the binary directly, manual setup:**

**For Zsh (recommended for macOS):**
```bash
# Add to your ~/.zshrc:
PROG=devkit
source <(curl -s https://raw.githubusercontent.com/Layr-Labs/devkit-cli/main/autocomplete/zsh_autocomplete)

exec zsh
```

**For Bash:**
```bash
# Add to your ~/.bashrc or ~/.bash_profile:
PROG=devkit
source <(curl -s https://raw.githubusercontent.com/Layr-Labs/devkit-cli/main/autocomplete/bash_autocomplete)

source ~/.bashrc
```

**For local development/testing:**
```bash
# If you have the devkit-cli repo locally
cd /path/to/devkit-cli
PROG=devkit source autocomplete/zsh_autocomplete  # for zsh
PROG=devkit source autocomplete/bash_autocomplete # for bash
```

After setup, you can use tab completion:
```bash
devkit <TAB>          # Shows: avs, keystore, version
devkit avs <TAB>      # Shows: create, config, context, build, devnet, run, call, release, template
devkit avs cr<TAB>    # Completes to: devkit avs create
```

---

## 🚧 Step-by-Step Guide

### 1️⃣ Create a New AVS Project (`devkit avs create`)

Sets up a new AVS project with the recommended structure, configuration files, and boilerplate code. This helps you get started quickly without needing to manually organize files or determine a layout. Details:

* Initializes a new project based on the default Hourglass task-based architecture in Go. Refer to [here](https://github.com/Layr-Labs/hourglass-avs-template?tab=readme-ov-file#what-is-hourglass) for details on the Hourglass architecture.
* Generates boilerplate code and default configuration.

Projects are created by default in the current directory from where the below command is called.

```bash
devkit avs create my-avs-project ./
cd my-avs-project
# If dependencies were installed during the creation process, you will need to source your bash/zsh profile:
#  - if you use bashrc
source ~/.bashrc
#  - if you use bash_profile
source ~/.bash_profile
#  - if you use zshrc
source ~/.zshrc
#  - if you use zprofile
source ~/.zprofile
```

> Note: Projects are created with a specific template version. You can view your current template version with `devkit avs template info` and upgrade later using `devkit avs template upgrade`.

> [!IMPORTANT]
> All subsequent `devkit avs` commands must be run from the root of your AVS project—the directory containing the [config](https://github.com/Layr-Labs/devkit-cli/tree/main/config) folder. The `config` folder contains the base `config.yaml` with the `contexts` folder which houses the respective context yaml files, example `devnet.yaml`.

<!-- Put in section about editing main.go file to replace comments with your actual business logic
-->

### 2️⃣ Implement Your AVS Task Logic (`main.go`)
After scaffolding your project, navigate into the project directory and begin implementing your AVS-specific logic. The core logic for task validation and execution lives in the `main.go` file inside the cmd folder:

```bash
cd my-avs-project/cmd
```

Within `main.go`, you'll find two critical methods on the `TaskWorker` type:
- **`HandleTask(*TaskRequest)`**  
  This is where you implement your AVS's core business logic. It processes an incoming task and returns a `TaskResponse`. Replace the placeholder comment with the actual logic you want to run during task execution.

- **`ValidateTask(*TaskRequest)`**  
  This method allows you to pre-validate a task before executing it. Use this to ensure your task meets your AVS's criteria (e.g., argument format, access control, etc.).

These functions will be invoked automatically when using `devkit avs call`, enabling you to quickly test and iterate on your AVS logic.

> **💡 Tip:**  
> You can add logging inside these methods using the `tw.logger.Sugar().Infow(...)` lines to debug and inspect task input and output during development.

### 3️⃣ Set RPC Endpoint URL

Set the `FORK_URL` values to a **Holesky** RPC **archive node** endpoint URL. This endpoint is needed to enable forking of the testnet state to your local environment. The endpoint will be used to fork the chain state to your local environment (devnet) for testing. Please note the following important details:
- Only the **Holesky** testnet is supported at this time.
- The RPC endpoint should be an **archive** node, not a _full_ node. More context is available [here](https://www.quicknode.com/guides/infrastructure/node-setup/ethereum-full-node-vs-archive-node).
- For initial testing purposes we recommend setting both `FORK_URL` values to the same endpoint URL.

```bash
cp .env.example .env
# edit `.env` and set your L1_FORK_URL and L2_FORK_URL to point to your RPC endpoint
```

You are welcome to use any reliable RPC provider (e.g. QuickNode, Alchemy).



### 4️⃣ Build Your AVS (`devkit avs build`)

Compiles your AVS contracts and offchain binaries. Required before running a devnet or simulating tasks to ensure all components are built and ready.

* Compiles smart contracts using Foundry.
* Builds operator, aggregator, and AVS logic binaries.

Ensure you're in your project directory before running:

```bash
devkit avs build
```

### 5️⃣ Launch Local DevNet (`devkit avs devnet`)

Starts a local devnet to simulate the full AVS environment. This step deploys contracts, registers operators, and runs offchain infrastructure, allowing you to test and iterate without needing to interact with testnet or mainnet.

* Forks Ethereum holesky using a fork URL (provided by you) and a block number. These URLs CAN be set in the `config/context/devnet.yaml`, but we recommend placing them in a `.env` file which will take precedence over `config/context/devnet.yaml`. Please see `.env.example`.
* Automatically funds wallets (`operator_keys` and `submit_wallet`) if balances are below `10 ether`.
* Setup required `AVS` contracts.
* Register `AVS` and `Operators`.

In your project directory, run:

```bash
devkit avs devnet start
```

> [!IMPORTANT]
> Please ensure your Docker daemon is running before running this command.

DevNet management commands:

| Command | Description                                                             |
| ------- | -------------------------------------------                             |
| `start` | Start local Docker containers and contracts                             |
| `stop`  | Stop and remove containers from the AVS project   |
| `list`  | List active containers and their ports                                  |
| `stop --all`  | Stops all devkit devnet containers that are currently running                                  |
| `stop --project.name`  | Stops the specific project's devnet                                  |
| `stop --port`  | Stops the specific port e.g.: `stop --port 8545`                                  |

### 6️⃣ Simulate Task Execution (`devkit avs call`)

Triggers task execution through your AVS, simulating how a task would be submitted, processed, and validated. Useful for testing end-to-end behavior of your logic in a local environment.

* Simulate the full lifecycle of task submission and execution.
* Validate both off-chain and on-chain logic.
* Review detailed execution results.

Run this from your project directory:

```bash
devkit avs call --signature="(uint256,string)" args='(5,"hello")'
```

Optionally, submit tasks directly to the on-chain TaskMailBox contract via a frontend or another method for more realistic testing scenarios.

### 7️⃣ Publish AVS Release (`devkit avs release`)

Publishes your AVS release to the EigenLayer ReleaseManager contract, making it available for operators to upgrade to.

* Publishes multi-architecture container images to the registry(linux/amd64,linux/arm64)
* Publishes release artifacts to the ReleaseManager contract.

Before publishing a release, ensure you have:
1. Built your AVS with `devkit avs build`
2. A running devnet
3. Properly configured registry in your context

Run this from your project directory:
> [!IMPORTANT]
> The upgrade-by-time must be in the future. Operators will have until this timestamp to upgrade to the new version.
> Devnet must be running before publishing.

```bash
devkit avs release publish  --upgrade-by-time 1750000000
```

**Required Flags:**
- `--upgrade-by-time`: Unix timestamp by which operators must upgrade

**Optional Flags:**
- `--registry`: Registry for the release (defaults to context)

Example
```bash
devkit avs release publish \
  --upgrade-by-time <future-timestamp> \
  --registry <ghcr.io/avs-release-example>
```


---

## Optional Commands


### Configure Your AVS (`devkit avs config` & `devkit avs context`)

Configure both project-level and context-specific settings via the following files:

- **`config.yaml`**  
  Defines project-wide settings such as AVS name, version, and available context names.  
- **`contexts/<context>.yaml`**  
  Contains environment-specific settings for a given context (e.g., `devnet`), including the Ethereum fork URL, block height, operator keys, AVS keys, and other runtime parameters.

You can view or modify these configurations using the DevKit CLI or by editing the `config.yaml` or the `contexts/*.yaml` files manually.

---

> [!IMPORTANT]
> All `devkit avs` commands must be run from the **root of your AVS project** — the directory containing the `config` folder.

#### View current settings

- **Project-level**  
  ```bash  
  devkit avs config --list
  ```

- **Context-specific**  
  ```bash  
  devkit avs context --list  
  devkit avs context --context devnet --list  
  ```

#### Edit settings directly via CLI

- **Project-level**  
  ```bash  
  devkit avs config --edit  
  ```

- **Context-specific**  
  ```bash  
  devkit avs context --edit  
  devkit avs context --context devnet --edit  
  ```

#### Set values via CLI flags

- **Project-level**
  ```bash
  devkit avs config --set project.name="My new name" project.version="0.0.2"
  ```

- **Context-specific**
  ```bash
  devkit avs context --set operators.0.address="0xabc..." operators.0.ecdsa_key="0x123..."
  devkit avs context --context devnet --set operators.0.address="0xabc..." operators.0.ecdsa_key="0x123..."
  ```





### Start offchain AVS infrastructure (`devkit avs run`)

Run your offchain AVS components locally.

* Initializes the Aggregator and Executor Hourglass processes.

This step is optional. The devkit `devkit avs devnet start` command already starts these components. However, you may choose to run this separately if you want to start the offchain processes without launching a local devnet — for example, when testing against a testnet deployment.

> Note: Testnet support is not yet implemented, but this command is structured to support such workflows in the future.

```bash
devkit avs run
```

### Deploy AVS Contracts (`devkit avs deploy-contract`)

Deploy your AVS's onchain contracts independently of the full devnet setup.

This step is **optional**. The `devkit avs devnet start` command already handles contract deployment as part of its full setup. However, you may choose to run this command separately if you want to deploy contracts without launching a local devnet — for example, when preparing for a testnet deployment.

> Note: Testnet support is not yet implemented, but this command is structured to support such workflows in the future.

```bash
devkit avs deploy-contract
```

### Create Operator Keys (`devkit avs keystore`)
Create and read keystores for bn254 private keys using the CLI. 

- To create a keystore
```bash
devkit keystore create --key --path --password
```

- To read an existing keystore
```bash
devkit keystore read --path --password
```

**Flag Descriptions**
- **`key`**: Private key in BigInt format. Example: `5581406963073749409396003982472073860082401912942283565679225591782850437460` 
- **`path`**: Path to the json file that must also include the filename. Example: `./keystores/operator1.keystore.json`
- **`password`**: Password to encrypt/decrypt the keystore.

### Template Management (`devkit avs template`)

Manage your project templates to stay up-to-date with the latest features and improvements.

* View current template information
* Upgrade your project to a newer template version

Subcommands:

| Command | Description |
| ------- | ----------- |
| `info` | Display information about the current project template |
| `upgrade` | Upgrade project to a newer template version |

View template information:
```bash
devkit avs template info
```

Upgrade to a specific template version (`"latest"`, tag, branch, or commit hash):
```bash
devkit avs template upgrade --version v1.0.0
```

### 📖 Logging (`--verbose`)

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
## Upgrade Process


### Upgrading the Devkit CLI

To upgrade the Devkit CLI to the latest version, you can use the `devkit upgrade` command.

```bash
# installs the latest version of devkit 
devkit upgrade
```

To move to a specific release, find the [target release](https://github.com/Layr-Labs/devkit-cli/releases) you want to install and run:

```bash
devkit upgrade --version <target-version>
```

### Upgrading your template

To upgrade the template you created your project with (by calling `devkit avs create`) you can use the `devkit avs template` subcommands.

```bash
# installs the latest template version known to devkit
devkit avs template upgrade
```

**_View which version you're currently using_**

```bash
devkit avs template info

2025/05/22 14:42:36 Project template information:
2025/05/22 14:42:36   Project name: <your project>
2025/05/22 14:42:36   Template URL: https://github.com/Layr-Labs/hourglass-avs-template
2025/05/22 14:42:36   Version: v0.0.13
```

**_Upgrade to a newer version_**

To upgrade to a newer version you can run:

```bash
devkit avs template upgrade --version <version>
```

More often than not, you'll want to use the tag corresponding to your template's release. You may also provide a branch name or commit hash to upgrade to.

_Please consult your template's docs for further information on how the upgrade process works._

---

## Telemetry 

DevKit includes optional telemetry to help us improve the developer experience. We collect anonymous usage data about commands used, performance metrics, and error patterns - but never personal information, code content, or sensitive data.

### 🎯 First-Time Setup

When you first run DevKit, you'll see a telemetry consent prompt:

```
🎯 Welcome to EigenLayer DevKit!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 Help us improve DevKit by sharing anonymous usage data
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

We'd like to collect anonymous usage data to help us improve DevKit.

This includes:
  • Commands used (e.g., 'devkit avs create', 'devkit avs build')
  • Error counts and types (to identify common issues)
  • Performance metrics (command execution times)
  • System information (OS, architecture)

We do NOT collect:
  • Personal information
  • Private keys or sensitive data

You can change this setting anytime with:
  devkit telemetry --enable   # Enable telemetry
  devkit telemetry --disable  # Disable telemetry

Would you like to enable telemetry? [Y/n]:
```

Your choice is saved globally and will be inherited by all future projects.

#### 🤖 Non-Interactive Environments

For CI/CD pipelines and automated environments, DevKit provides several options:

**Enable telemetry without prompting:**
```bash
devkit --enable-telemetry avs create my-project 
```

**Disable telemetry without prompting:**
```bash
devkit --disable-telemetry avs create my-project 
```

**CI environments** (when `CI=true` environment variable is set):
- DevKit automatically detects CI environments and defaults to disabled telemetry
- No prompting occurs, preventing pipeline hangs
- You can still explicitly enable with `--enable-telemetry` if desired

**Non-interactive terminals:**
- DevKit detects when stdin is unavailable and skips prompting
- Defaults to disabled telemetry with informational messages

### 📊 What Data We Collect

**✅ We collect:**
- Command names (e.g., `devkit avs create`, `devkit avs build`)
- Success/failure rates and error types
- Command execution duration
- Operating system and architecture
- Anonymous project identifiers (UUIDs)

**❌ We do NOT collect:**
- Personal information or identifiable data
- Code content, file names, or project details
- Private keys, passwords, or sensitive data

### 🛠 Managing Telemetry Settings

#### Global Settings (affects all projects)

```bash
# Enable telemetry globally (new projects inherit this)
devkit telemetry --enable --global

# Disable telemetry globally  
devkit telemetry --disable --global

# Check global telemetry status
devkit telemetry --status --global
```

#### Project-Level Settings (current project only)

```bash
# Enable telemetry for current project only
devkit telemetry --enable

# Disable telemetry for current project only
devkit telemetry --disable

# Check current project telemetry status
devkit telemetry --status
```

### 📋 How Telemetry Precedence Works

1. **Project setting exists?** → Use project setting
2. **No project setting?** → Use global setting  
3. **No settings at all?** → Default to disabled

This means:
- You can set a global default for all projects
- Individual projects can override the global setting
- Existing projects keep their current settings when you change global settings

### 📁 Configuration Files

**Global config:** `~/.config/devkit/config.yaml`
```yaml
first_run: false
telemetry_enabled: true
```

**Project config:** `<project-dir>/.config.devkit.yml`
```yaml
project_uuid: "12345678-1234-1234-1234-123456789abc"
telemetry_enabled: true
```

### 🔄 Common Workflows

**Set global default for your organization:**
```bash
# Disable telemetry for all future projects
devkit telemetry --disable --global
```

**Override for a specific project:**
```bash
# In project directory - enable telemetry just for this project
cd my-avs-project
devkit telemetry --enable
```

**Check what's actually being used:**
```bash
# Shows both project and global settings for context
devkit telemetry --status
```


### 🏢 Enterprise Usage

For enterprise environments, you can:

1. **Set organization-wide defaults** by configuring global settings
2. **Override per-project** as needed for specific teams or compliance requirements
3. **Completely disable** telemetry with `devkit telemetry --disable --global`

The telemetry system respects both user choice and organizational policies.

## 🔧 Compatibility Notes
- **Linux**: Primarily tested on Debian/Ubuntu only.
- **macOS**: Supports both Intel and Apple Silicon

## 🤝 Contributing

Contributions are welcome! Please open an issue to discuss significant changes before submitting a pull request.

## Troubleshooting

If you want to debug any transaction failure, try using `--verbose` flag with the command, to get tx_hash in your logs.
 
## 🙋 Help (Support)
Please post any questions or concerns to the [Issues](https://github.com/Layr-Labs/devkit-cli/issues) tab in this repo. We will respond to your issue as soon as our team has capacity, however we are not yet able to offer an SLA for response times. Please do not use this project for Production, Mainnet, or time sensitive use cases at this time.

---

## For DevKit Maintainers: DevKit Release Process
To release a new version of the CLI, follow the steps below:
> Note: You need to have write permission to this repo to release a new version.

1. Checkout the main branch and pull the latest changes:
    ```bash
    git checkout main
    git pull origin main
    ```
2. In your local clone, create a new release tag using the following command:
    ```bash
    git tag v<version> -m "Release v<version>"
    ```
3. Push the tag to the repository using the following command:
    ```bash
    git push origin v<version>
    ```

4. This will automatically start the release process in the [GitHub Actions](https://github.com/Layr-Labs/eigenlayer-cli/actions/workflows/release.yml) and will create a draft release to the [GitHub Releases](https://github.com/Layr-Labs/eigenlayer-cli/releases) with all the required binaries and assets
5. Check the release notes and add any notable changes and publish the release
