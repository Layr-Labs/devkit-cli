package context

import (
	stdctx "context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Layr-Labs/devkit-cli/config/contexts"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/Layr-Labs/devkit-cli/pkg/common/output"
	"gopkg.in/yaml.v3"

	gethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/urfave/cli/v2"
)

var ChainIDFromRPC = getChainIDFromRPC

var CreateContextCommand = &cli.Command{
	Name:      "create",
	Usage:     "Create a new context",
	ArgsUsage: "devkit avs context create <name>",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:  "context",
			Usage: "Select the context to work over",
		},
		&cli.BoolFlag{
			Name:  "overwrite",
			Usage: "Overwrite if the context already exists",
		},
		&cli.BoolFlag{
			Name:  "use",
			Usage: "Set as current context",
			Value: true,
		},
		&cli.StringFlag{
			Name:  "l1-rpc-url",
			Usage: "Set the L1 RPC URL for the context",
		},
		&cli.StringFlag{
			Name:  "l2-rpc-url",
			Usage: "Set the L2 RPC URL for the context",
		},
		&cli.StringFlag{
			Name:  "deployer-private-key",
			Usage: "Deployer private key for contracts on L1/L2",
		},
		&cli.StringFlag{
			Name:  "app-private-key",
			Usage: "Application owner private key used by AVS",
		},
		&cli.StringFlag{
			Name:  "avs-private-key",
			Usage: "AVS operator private key",
		},
		&cli.StringFlag{
			Name:  "avs-metadata-url",
			Usage: "Public JSON metadata URL for AVS",
		},
		&cli.StringFlag{
			Name:  "registrar-address",
			Usage: "Registrar contract address",
		},
	}, common.GlobalFlags...),
	Action: contextCreateAction,
}

func contextCreateAction(cCtx *cli.Context) error {
	logger := common.LoggerFromContext(cCtx)

	// Use flag provided name
	name := cCtx.String("context")

	// Get context name from arg
	if name == "" {
		name = cCtx.Args().Get(0)
	}

	// If no context is provided show help
	if name == "" {
		return cli.ShowSubcommandHelp(cCtx)
	}

	// Locate the context directory
	cntxDir := filepath.Join("config", "contexts")

	// Guard existence early
	if err := ensureContextCreatable(cntxDir, name, cCtx.Bool("overwrite")); err != nil {
		return err
	}

	logger.Info("Creating new context: %s", name)

	// If re-creating devnet, copy latest and exit early
	if name == "devnet" {
		// Path to the context.yaml file
		ctxPath := filepath.Join(cntxDir, fmt.Sprintf("%s.yaml", name))

		// Pull the latest context and set name
		content := contexts.ContextYamls[contexts.LatestVersion]
		entryName := fmt.Sprintf("%s.yaml", name)

		// Write the new context
		err := os.WriteFile(ctxPath, []byte(content), 0644)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", entryName, err)
		}

		// Set the current context in config
		if cCtx.Bool("use") {
			err := setCurrentContext(name)
			if err != nil {
				return fmt.Errorf("failed to set current context %s: %w", entryName, err)
			}
		}

		// Load the new context
		data, name, err := common.LoadRawContext(name)
		if err != nil {
			return fmt.Errorf("failed to read context %s: %w", entryName, err)
		}
		var ctxDoc common.ContextConfig
		if err := yaml.Unmarshal(data, &ctxDoc); err != nil {
			return fmt.Errorf("failed to parse context %q: %w", name, err)
		}

		// Log creation
		logContextCreated(logger, cntxDir, name, &ctxDoc, cCtx.Bool("use"))

		return nil
	}

	// L1
	l1RPCURL, err := getL1RPCURL(cCtx)
	if err != nil {
		return err
	}
	l1ChainID, err := ChainIDFromRPC(l1RPCURL, logger)
	if err != nil {
		return err
	}

	// L2
	l2RPCURL, err := getL2RPCURL(cCtx)
	if err != nil {
		return err
	}
	l2ChainID, err := ChainIDFromRPC(l2RPCURL, logger)
	if err != nil {
		return err
	}

	// Keys and AVS basics
	deployerKey, err := getDeployerKey(cCtx)
	if err != nil {
		return err
	}
	appKey, err := getAppKey(cCtx)
	if err != nil {
		return err
	}
	avsCfg, err := getAVSSetup(cCtx)
	if err != nil {
		return err
	}

	// Build context
	ctxDoc := CreateContext(
		name,
		int(l1ChainID.Uint64()), l1RPCURL,
		int(l2ChainID.Uint64()), l2RPCURL,
		deployerKey, appKey, avsCfg,
	)

	// Persist and optionally make current
	if err := saveContext(cntxDir, name, ctxDoc, cCtx.Bool("use"), logger); err != nil {
		return err
	}

	// Load the context to update addresses with zeus
	yamlPath, rootNode, contextNode, contextName, err := common.LoadContext(name)
	if err != nil {
		logger.Title("Could not load context YAML for Zeus update: %v", err)
	} else {
		if err := common.UpdateContextWithZeusAddresses(cCtx.Context, logger, contextNode, contextName); err != nil {
			logger.Info("Failed to fetch addresses from Zeus: %v", err)
			logger.Info("Continuing with addresses from config...")
		} else {
			logger.Info("Successfully updated context with addresses from Zeus")
			if err := common.WriteYAML(yamlPath, rootNode); err != nil {
				return fmt.Errorf("failed to save updated context: %v", err)
			}
		}
	}

	logContextCreated(logger, cntxDir, name, ctxDoc, cCtx.Bool("use"))
	return nil
}

func ensureContextCreatable(cntxDir, ctxName string, overwrite bool) error {
	ctxPath := filepath.Join(cntxDir, fmt.Sprintf("%s.yaml", ctxName))
	if err := os.MkdirAll(filepath.Dir(ctxPath), 0o755); err != nil {
		return fmt.Errorf("failed to make contexts dir: %w", err)
	}
	if _, err := os.Stat(ctxPath); err == nil && !overwrite {
		return fmt.Errorf("context '%s' already exists. Use --overwrite to replace it", ctxName)
	}
	return nil
}

func getL1RPCURL(cCtx *cli.Context) (string, error) {
	l1RPCURL := cCtx.String("l1-rpc-url")
	if l1RPCURL == "" {
		url, err := output.InputString(
			"Enter L1 RPC URL",
			"The RPC endpoint URL for the L1 network (e.g., http://localhost:8545)",
			"",
			validateRPCURL,
		)
		if err != nil {
			return "", fmt.Errorf("failed to get L1 RPC URL: %w", err)
		}
		return url, nil
	}
	if err := validateRPCURL(l1RPCURL); err != nil {
		return "", fmt.Errorf("invalid L1 RPC URL: %w", err)
	}
	return l1RPCURL, nil
}

func getChainIDFromRPC(rpcURL string, logger iface.Logger) (*big.Int, error) {
	logger.Debug("Retrieving Chain ID from RPC: %s", rpcURL)
	ethClient, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}
	defer ethClient.Close()

	chainID, err := ethClient.ChainID(stdctx.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	logger.Debug("Connected to RPC: %s", chainID.String())
	return chainID, nil
}

func getL2RPCURL(cCtx *cli.Context) (string, error) {
	l2RPCURL := cCtx.String("l2-rpc-url")
	if l2RPCURL == "" {
		url, err := output.InputString(
			"Enter L2 RPC URL",
			"The RPC endpoint URL for the L2 network (e.g., http://localhost:9545)",
			"",
			validateRPCURL,
		)
		if err != nil {
			return "", fmt.Errorf("failed to get L2 RPC URL: %w", err)
		}
		return url, nil
	}
	if err := validateRPCURL(l2RPCURL); err != nil {
		return "", fmt.Errorf("invalid L2 RPC URL: %w", err)
	}
	return l2RPCURL, nil
}

func getDeployerKey(cCtx *cli.Context) (string, error) {
	deployerPrivateKey := cCtx.String("deployer-private-key")
	if deployerPrivateKey == "" {
		val, err := output.InputString(
			"Enter a funded Deployer Private Key",
			"The private key used to deploy L1/L2 contracts (e.g., 0xac09...)",
			"",
			validatePrivateKey,
		)
		if err != nil {
			return "", fmt.Errorf("failed to get Deployer key: %w", err)
		}
		deployerPrivateKey = val
	}
	if err := validatePrivateKey(deployerPrivateKey); err != nil {
		return "", fmt.Errorf("invalid Deployer private key: %w", err)
	}
	return "0x" + strip0x(deployerPrivateKey), nil
}

func getAppKey(cCtx *cli.Context) (string, error) {
	appPrivateKey := cCtx.String("app-private-key")
	if appPrivateKey == "" {
		val, err := output.InputString(
			"Enter a funded App private key",
			"The private key used to deploy and control AVS contracts",
			"",
			validatePrivateKey,
		)
		if err != nil {
			return "", fmt.Errorf("failed to get App private key: %w", err)
		}
		appPrivateKey = val
	}
	if err := validatePrivateKey(appPrivateKey); err != nil {
		return "", fmt.Errorf("invalid App private key: %w", err)
	}
	return "0x" + strip0x(appPrivateKey), nil
}

func getAVSSetup(cCtx *cli.Context) (*common.AvsConfig, error) {
	privateKey, err := getAVSPrivateKey(cCtx)
	if err != nil {
		return nil, err
	}
	address, err := derivePublicKey(privateKey)
	if err != nil {
		return nil, err
	}
	metadataURL, err := getAVSMetadataURL(cCtx)
	if err != nil {
		return nil, err
	}

	cfg := &common.AvsConfig{
		Address:          address,
		AVSPrivateKey:    privateKey,
		MetadataUri:      metadataURL,
		RegistrarAddress: "",
	}

	return cfg, nil
}

func getAVSPrivateKey(cCtx *cli.Context) (string, error) {
	pk := cCtx.String("avs-private-key")
	if pk == "" {
		val, err := output.InputString(
			"Enter a funded AVS private key",
			"64 hex characters with optional 0x",
			"",
			validatePrivateKey,
		)
		if err != nil {
			return "", fmt.Errorf("failed to get AVS private key: %w", err)
		}
		pk = val
	}
	if err := validatePrivateKey(pk); err != nil {
		return "", fmt.Errorf("invalid AVS private key: %w", err)
	}
	return "0x" + strip0x(pk), nil
}

func derivePublicKey(pkHex string) (string, error) {
	raw := strip0x(pkHex)
	d, err := gethcrypto.HexToECDSA(raw)
	if err != nil {
		return "", fmt.Errorf("invalid private key material: %w", err)
	}
	pub := d.Public().(*ecdsa.PublicKey)
	addr := gethcrypto.PubkeyToAddress(*pub)
	return strings.ToLower(addr.Hex()), nil
}

func getAVSMetadataURL(cCtx *cli.Context) (string, error) {
	u := cCtx.String("avs-metadata-url")
	if u == "" {
		val, err := output.InputString(
			"Enter AVS metadata URL",
			"Public URL to a JSON metadata document, for example https://my-org.com/avs/metadata.json",
			"",
			validateMetadataURL,
		)
		if err != nil {
			return "", fmt.Errorf("failed to get AVS metadata URL: %w", err)
		}
		return val, nil
	}
	if err := validateMetadataURL(u); err != nil {
		return "", fmt.Errorf("invalid AVS metadata URL: %w", err)
	}
	return u, nil
}

func CreateContext(
	name string,
	l1ChainID int, l1RPCURL string,
	l2ChainID int, l2RPCURL string,
	deployerKey string, appKey string, avs *common.AvsConfig,
) *common.ContextConfig {

	return &common.ContextConfig{
		Version: contexts.LatestVersion,
		Context: common.ChainContextConfig{
			Name: name,
			Chains: map[string]common.ChainConfig{
				"l1": {ChainID: l1ChainID, RPCURL: l1RPCURL},
				"l2": {ChainID: l2ChainID, RPCURL: l2RPCURL},
			},
			DeployerPrivateKey:    deployerKey,
			AppDeployerPrivateKey: appKey,
			Avs:                   *avs,
			// Place an empty artifact record to be updated on release
			Artifact: &common.ArtifactConfig{
				ArtifactId: "",
				Component:  "",
				Digest:     "",
				Registry:   "",
				Version:    "",
			},
		},
	}
}

func saveContext(cntxDir, name string, ctx *common.ContextConfig, setCurrent bool, logger iface.Logger) error {
	// write config/contexts/<name>.yaml
	out := struct {
		Version string                    `yaml:"version"`
		Context common.ChainContextConfig `yaml:"context"`
	}{
		Version: ctx.Version,
		Context: ctx.Context,
	}

	ctxPath := filepath.Join(cntxDir, name+".yaml")
	data, err := yaml.Marshal(out)
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}
	if err := os.WriteFile(ctxPath, data, 0o600); err != nil {
		return fmt.Errorf("write context file: %w", err)
	}

	logger.Info("\nWrote context file: %s", ctxPath)

	// Set the current context in config
	if setCurrent {
		return setCurrentContext(name)
	}

	return nil
}

func setCurrentContext(name string) error {
	// Load base config, update project.context, persist
	base, err := common.LoadBaseConfigYaml()
	if err != nil {
		return fmt.Errorf("load base config: %w", err)
	}
	base.Config.Project.Context = name
	basePath := filepath.Join(common.DefaultConfigWithContextConfigPath, common.BaseConfig)
	baseBytes, err := yaml.Marshal(base)
	if err != nil {
		return fmt.Errorf("marshal base config: %w", err)
	}
	if err := os.WriteFile(basePath, baseBytes, 0o600); err != nil {
		return fmt.Errorf("write base config: %w", err)
	}

	return nil
}

func logContextCreated(logger iface.Logger, cntxDir, name string, ctx *common.ContextConfig, setCurrent bool) {
	l1 := ctx.Context.Chains["l1"]
	l2 := ctx.Context.Chains["l2"]
	avs := ctx.Context.Avs

	logger.Title("Context created:")
	logger.Info(" - name: %s", name)
	logger.Info(" - L1 chain: %s (%d)", l1.RPCURL, l1.ChainID)
	logger.Info(" - L2 chain: %s (%d)", l2.RPCURL, l2.ChainID)

	if setCurrent {
		logger.Info(" - Global context set to '%s'", name)
	}

	logger.Title("Keys:")
	logger.Info(" - deployerPrivateKey: %s", ctx.Context.DeployerPrivateKey)
	logger.Info(" - appPrivateKey: %s", ctx.Context.AppDeployerPrivateKey)

	logger.Title("AVS:")
	logger.Info(" - address: %s", avs.Address)
	logger.Info(" - metadataUrl: %s", avs.MetadataUri)
	logger.Info(" - registrarAddress: %s", avs.RegistrarAddress)

	logger.Title("Context successfully created at %s", filepath.Join(cntxDir, fmt.Sprintf("%s.yaml", name)))
	logger.Info("  - To view your new context call: `devkit avs context --list %s`", name)
	logger.Info("  - To edit your new context call: `devkit avs context --edit %s`", name)
	logger.Info("")
}

func strip0x(s string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s)), "0x")
}

func validateRPCURL(input string) error {
	if input == "" {
		return fmt.Errorf("RPC URL cannot be empty")
	}
	urlPattern := regexp.MustCompile(`^(https?|wss?)://`)
	if !urlPattern.MatchString(input) {
		return fmt.Errorf("RPC URL must start with http://, https://, ws://, or wss://")
	}
	return nil
}

func validatePrivateKey(input string) error {
	if input == "" {
		return fmt.Errorf("private key cannot be empty")
	}
	keyPattern := regexp.MustCompile(`^(0x)?[0-9a-fA-F]{64}$`)
	if !keyPattern.MatchString(input) {
		return fmt.Errorf("private key must be 64 hex characters, optionally prefixed with 0x")
	}
	return nil
}

func validateMetadataURL(input string) error {
	if input == "" {
		return fmt.Errorf("metadata URL cannot be empty")
	}
	urlPattern := regexp.MustCompile(`^https?://`)
	if !urlPattern.MatchString(input) {
		return fmt.Errorf("metadata URL must start with http:// or https://")
	}
	if !regexp.MustCompile(`\.json(\?.*)?$`).MatchString(input) {
		return fmt.Errorf("metadata URL should point to a .json resource")
	}
	return nil
}
