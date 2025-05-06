# EigenLayer Development Kit (DevKit)

A CLI tool for developing and managing EigenLayer AVS (Autonomous Verifiable Services) projects.

---

## 🚀 Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/engine/install/)
- [Go](https://go.dev/doc/install)
- [Foundry](https://book.getfoundry.sh/getting-started/installation)
- [make](https://formulae.brew.sh/formula/make)

#### Setup to fetch private go modules

To ensure you can fetch private Go modules hosted on GitHub (needed before the template dependency repos are live):

1.  **Ensure SSH Key is Added to GitHub:** Verify that you have an SSH key associated with your GitHub account. You can find instructions [here](https://docs.github.com/en/authentication/connecting-to-github-with-ssh/adding-a-new-ssh-key-to-your-github-account).
2.  **Repository Access:** Confirm with EigenLabs that your GitHub account has been granted access to the necessary private repositories (e.g., for preview features or specific AVS components).
3.  **Configure Git URL Rewrite:** Run the following command in your terminal to instruct Git to use SSH instead of HTTPS for Eigenlabs repositories:
    ```bash
    git config --global url."ssh://git@github.com/Layr-Labs/".insteadOf "https://github.com/Layr-Labs/"
    ```

If you are on OSX, ensure that your `~/.ssh/config` file does not contain the line `UseKeychain yes`, as it can interfere with SSH agent forwarding or other SSH functionalities needed for fetching private modules. If it exists, you may need to comment it out or remove it.


```bash
# Clone and build
git clone https://github.com/Layr-Labs/devkit-cli
cd devkit-cli

# Install the CLI
make install

# Or build manually
go build -o devkit ./cmd/devkit

# Get started
devkit --help
```

## Demo flow
```bash
# If not already, clone it
git clone https://github.com/Layr-Labs/devkit-cli
cd devkit-cli

# If not already, pull the latest commit
git pull origin main

# Note that you have to run the create command from repository directory
devkit avs create my-hourglass-project  # by default pick task arch and go lang
OR
devkit avs create --overwrite my-existing-hourglass-project

# Once you have a project directory, following commands should be run from the project directory you created.
devkit avs build

devkit avs devnet start

devkit avs run
```


## 🛠️ Development Workflow

```bash
make help      # Show all available dev commands
make build     # Build CLI binary
make tests     # Run all unit tests
make lint      # Run linter and static checks
```


## 💻 Core DevKit Commands
> [!IMPORTANT]  
> All <code>devkit avs</code> commands(except `devkit avs create`) must be run from the root of your AVS project — the directory that contains the <code>eigen.toml</code> file.  
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
The devnet consists of [eigenlayer-contracts-1.3.0](https://github.com/Layr-Labs/eigenlayer-contracts/tree/v1.3.0) deployed on top of a fresh anvil state.
We automatically fund the wallets(`operator_keys` and `submit_wallet`) used in the `eigen.toml` if balance is low(< `10 ether`).

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

### Config
We autogenerate a default config file called `eigen.toml` in the avs project directory. 

> [!Warning]
> These commands must be run from the directory of the project you created using `devkit avs create`.
#### List the current config
This commands lists the current configuration including `eigen.toml` , telemetry status etc.

```bash
devkit avs config 
```
Or 
```bash
devkit avs config --list
```

#### Edit the config
There are 2 ways to edit `eigen.toml` config of the respective avs project.

##### Option 1
This will allow to edit the config in a text editor.
```bash
devkit avs config --edit
```

##### Option 2
Manually edit the config in `eigen.toml`.

## ⚙️ Global Options

| Flag             | Description            |
|------------------|------------------------|
| `--verbose`, `-v`| Enable verbose logging |
| `--help`, `-h`   | Show help output       |


## 💡 Example Usage
```bash
# Scaffold a new AVS named MyAVS
devkit avs create MyAVS --lang go

# Start a local devnet
devkit avs devnet start

# Stop the devnet
devkit avs devnet stop
```

## Logging
- To enable persistent logging , you can set the verbosity under the key `[log]` in `eigen.toml`. By default it's set to `debug`.
```toml
[log]
level = "debug" # valid options: "info", "debug", "warn", "error"
```

- You can also use `--verbose` flag with the respective command, example:
```bash
devkit --verbose avs build
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

## 🤝 Contributing
Pull requests are welcome! For major changes, open an issue first to discuss what you would like to change.
