# EigenLayer Development Kit (DevKit) 🚀

## ⚠️ Disclaimer: Closed Alpha Not Production Ready
EigenLayer DevKit is currently in a closed alpha stage and is intended strictly for local experimentation and development. It has not been audited, and should not be used for use in any live environment, including public testnets or mainnet. Users are strongly discouraged from pushing generated projects to remote repositories without reviewing and sanitizing sensitive configuration files (e.g. devnet.yaml), which may contain private keys or other sensitive material.

**A CLI toolkit for developing, testing, and managing EigenLayer Autonomous Verifiable Services (AVS).**

EigenLayer DevKit streamlines AVS development, enabling you to quickly scaffold projects, compile contracts, run local networks, and simulate tasks with ease.

![EigenLayer DevKit User Flow](assets/devkit-user-flow.png)



## 🌟 Key Commands Overview

| Command      | Description                              |
| ------------ | ---------------------------------------- |
| `avs create` | Scaffold a new AVS project               |
| `avs config` | Configure your AVS (`config/config.yaml`,`config/devnet.yaml`...)        |
| `avs build`  | Compile AVS smart contracts and binaries |
| `avs devnet` | Manage local development network         |
| `avs call`   | Simulate AVS task execution locally      |

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

During this Private Preview, you'll need access to private Go modules hosted on GitHub:

1. **Add SSH Key to GitHub:** Ensure your SSH key is associated with your GitHub account ([instructions](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/adding-a-new-ssh-key-to-your-github-account)).
2. **Verify Repository Access:** Confirm with EigenLabs support that your account has access to necessary private repositories.

---

## 🚧 Step-by-Step Guide

### 1️⃣ Create a New AVS Project (`avs create`)

Sets up a new AVS project with the recommended structure, configuration files, and boilerplate code. This helps you get started quickly without needing to manually organize files or determine a layout. Details:

* Initializes a new project based on the default Hourglass task-based architecture in Go.
* Generates boilerplate code and default configuration.

Projects are created by default in the current directory from where the below command is called.

```bash
devkit avs create my-avs-project
cd my-avs-project
```

> \[!IMPORTANT]
> All subsequent `devkit avs` commands must be run from the root of your AVS project—the directory containing the [config](https://github.com/Layr-Labs/devkit-cli/tree/main/config) folder. The `config` folder contains the base `config.yaml` with the `contexts` folder which houses the respective context yaml files, example `devnet.yaml`.

### 2️⃣ Configure Your AVS (`avs config`)

Before running your AVS, you’ll need to configure both project-level and environment-specific settings. This is done through two configuration files:

- **`config.yaml`**: Defines project-wide settings such as AVS name and context names.
- **`contexts/devnet.yaml`**: Contains environment-specific settings for your a given context (i.e. devnet), including the Ethereum fork url, block height, operator keys, AVS keys, and other runtime parameters.

You can view or modify these configurations using the DevKit CLI or by editing the files manually.

View current settings via CLI:

```bash
devkit avs config
```

Edit settings directly via CLI:

```bash
devkit avs config --edit --path <path to the config.yaml or contexts/devnet.yaml file>
```

Alternatively, manually edit the config files in the text editor of your choice.

> \[!IMPORTANT]
> These commands must be run from your AVS project's root directory.

### 3️⃣ Build Your AVS

Compiles your AVS contracts and offchain binaries. Required before running a devnet or simulating tasks to ensure all components are built and ready.

* Compiles smart contracts using Foundry.
* Builds operator, aggregator, and AVS logic binaries.

Ensure you're in your project directory before running:

```bash
devkit avs build
```

### 4️⃣ Launch Local DevNet

Starts a local devnet to simulate the full AVS environment. This step deploys contracts, registers operators, and runs offchain infrastructure, allowing you to test and iterate without needing to interact with testnet or mainnet.

* Forks Ethereum mainnet using a fork URL (provided by you) and a block number.
* Automatically funds wallets (`operator_keys` and `submit_wallet`) if balances are below `10 ether`.
* Setup required `AVS` contracts.
* Register `AVS` and `Operators`.

> Note: You must provide a fork URL that forks from Ethereum mainnet. You can use any popular RPC provider such as QuickNode or Alchemy.

This step is essential for simulating your AVS environment in a fully self-contained way, enabling fast iteration on your AVS business logic without needing to deploy to testnet/mainnet or coordinate with live operators.

Run this from your project directory:

```bash
devkit avs devnet start
```

> \[!IMPORTANT]
> Please ensure your Docker daemon is running before running this command.

DevNet management commands:

| Command | Description                                                             |
| ------- | -------------------------------------------                             |
| `start` | Start local Docker containers and contracts                             |
| `stop`  | Stop and remove container from the avs project this command is called   |
| `list`  | List active containers and their ports                                  |
| `stop --all`  | Stops all devkit devnet containers that are currently currening                                  |
| `stop --project.name`  | Stops the specific project's devnet                                  |
| `stop --port`  | Stops the specific port .ex: `stop --port 8545`                                  |

### 5️⃣ Simulate Task Execution (`avs call`)

Triggers task execution through your AVS, simulating how a task would be submitted, processed, and validated. Useful for testing end-to-end behavior of your logic in a local environment.

* Simulate the full lifecycle of task submission and execution.
* Validate both off-chain and on-chain logic.
* Review detailed execution results.

Run this from your project directory:

```bash
devkit avs call
```

Optionally, submit tasks directly to the on-chain TaskMailBox contract via a frontend or another method for more realistic testing scenarios.

---

## Optional Commands

### Start offchain AVS infrastructure (`avs run`)

Run your offchain AVS components locally.

* Initializes the Aggregator and Executor Hourglass processes.

This step is optional. The devkit `devkit avs devnet start` command already starts these components. However, you may choose to run this separately if you want to start the offchain processes without launching a local devnet — for example, when testing against a testnet deployment.

> Note: Testnet support is not yet implemented, but this command is structured to support such workflows in the future.

```bash
devkit avs run
```

### Deploy AVS Contracts (`avs deploy-contract`)

Deploy your AVS’s onchain contracts independently of the full devnet setup.

This step is **optional**. The `devkit avs devnet start` command already handles contract deployment as part of its full setup. However, you may choose to run this command separately if you want to deploy contracts without launching a local devnet — for example, when preparing for a testnet deployment.

> Note: Testnet support is not yet implemented, but this command is structured to support such workflows in the future.

```bash
devkit avs deploy-contract
```

### Create Operator Keys (`avs keystore`)
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
- **`key`**: Private key in BigInt format . Example: `5581406963073749409396003982472073860082401912942283565679225591782850437460` 
- **`path`**: Path to the json file. It needs to include the filename . Example: `./keystores/operator1.keystore.json`
- **`password`**: Password to encrypt/decrypt the keystore.

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

## 🤝 Contributing

Contributions are welcome! Please open an issue to discuss significant changes before submitting a pull request.

---

## For DevKit Maintainers: DevKit Release Process
To release a new version of the CLI, follow the steps below:
> Note: You need to have write permission to this repo to release new version

1. Checkout the master branch and pull the latest changes:
    ```bash
    git checkout master
    git pull origin master
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
