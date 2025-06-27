package common

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Layr-Labs/crypto-libs/pkg/bn254"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"
	"github.com/Layr-Labs/devkit-cli/pkg/common/logger"
	"github.com/Layr-Labs/devkit-cli/pkg/common/progress"
	"github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/IKeyRegistrar"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

// loggerContextKey is used to store the logger in the context
type loggerContextKey struct{}

// progressTrackerContextKey is used to store the progress tracker in the context
type progressTrackerContextKey struct{}

// RexExp to match semver strings
var semverRegex = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)

// IsVerboseEnabled checks if either the CLI --verbose flag is set,
// or config.yaml has [log] level = "debug"
func IsVerboseEnabled(cCtx *cli.Context, cfg *ConfigWithContextConfig) bool {
	// Check CLI flag
	if cCtx.Bool("verbose") {
		return true
	}

	// Check config.yaml config
	// level := strings.ToLower(strings.TrimSpace(cfg.Log.Level))  // TODO(nova): Get log level debug from config.yaml also . For now only using the cli flag
	// return level == "debug"
	return true
}

// GetLoggerFromCLIContext creates a logger based on the CLI context
// It checks the verbose flag and returns the appropriate logger
func GetLoggerFromCLIContext(cCtx *cli.Context) (iface.Logger, iface.ProgressTracker) {
	verbose := cCtx.Bool("verbose")
	return GetLogger(verbose)
}

// Get logger for the env we're in
func GetLogger(verbose bool) (iface.Logger, iface.ProgressTracker) {

	var log iface.Logger
	var tracker iface.ProgressTracker

	if progress.IsTTY() {
		log = logger.NewLogger(verbose)
		tracker = progress.NewTTYProgressTracker(10, os.Stdout)
	} else {
		log = logger.NewZapLogger(verbose)
		tracker = progress.NewLogProgressTracker(10, log)
	}

	return log, tracker
}

// isCI checks if the code is running in a CI environment like GitHub Actions.
func isCI() bool {
	return os.Getenv("CI") == "true"
}

// WithLogger stores the logger in the context
func WithLogger(ctx context.Context, logger iface.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// WithProgressTracker stores the progress tracker in the context
func WithProgressTracker(ctx context.Context, tracker iface.ProgressTracker) context.Context {
	return context.WithValue(ctx, progressTrackerContextKey{}, tracker)
}

// LoggerFromContext retrieves the logger from the context
// If no logger is found, it returns a non-verbose logger as fallback
func LoggerFromContext(ctx context.Context) iface.Logger {
	if logger, ok := ctx.Value(loggerContextKey{}).(iface.Logger); ok {
		return logger
	}
	// Fallback to non-verbose logger if not found in context
	log, _ := GetLogger(false)
	return log
}

// ProgressTrackerFromContext retrieves the progress tracker from the context
// If no tracker is found, it returns a non-verbose tracker as fallback
func ProgressTrackerFromContext(ctx context.Context) iface.ProgressTracker {
	if tracker, ok := ctx.Value(progressTrackerContextKey{}).(iface.ProgressTracker); ok {
		return tracker
	}
	// Fallback to non-verbose tracker if not found in context
	_, tracker := GetLogger(false)
	return tracker
}

// ParseETHAmount parses ETH amount strings like "5ETH", "10.5ETH", "1000000000000000000" (wei)
// Returns the amount in wei as *big.Int
func ParseETHAmount(amountStr string) (*big.Int, error) {
	if amountStr == "" {
		return nil, fmt.Errorf("amount string is empty")
	}

	// Remove any whitespace
	amountStr = strings.TrimSpace(amountStr)

	// Check if it ends with "ETH"
	if strings.HasSuffix(strings.ToUpper(amountStr), "ETH") {
		// Remove the "ETH" suffix (case insensitive)
		ethIndex := strings.LastIndex(strings.ToUpper(amountStr), "ETH")
		numericPart := strings.TrimSpace(amountStr[:ethIndex])

		// Parse the numeric part as float64 to handle decimals like "1.5ETH"
		ethAmount, err := strconv.ParseFloat(numericPart, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid ETH amount '%s': %w", numericPart, err)
		}

		// Convert ETH to wei (multiply by 10^18)
		// Use big.Float to handle the large numbers properly
		ethBig := big.NewFloat(ethAmount)
		weiPerEth := big.NewFloat(1e18)
		weiBig := new(big.Float).Mul(ethBig, weiPerEth)

		// Convert to big.Int
		weiInt, _ := weiBig.Int(nil)
		return weiInt, nil
	}

	// If no "ETH" suffix, assume it's already in wei
	weiAmount := new(big.Int)
	if _, ok := weiAmount.SetString(amountStr, 10); !ok {
		return nil, fmt.Errorf("invalid wei amount '%s'", amountStr)
	}

	return weiAmount, nil
}

// ImpersonateAccount enables impersonation of an account on Anvil
func ImpersonateAccount(client *rpc.Client, address common.Address) error {
	var result interface{}
	err := client.Call(&result, "anvil_impersonateAccount", address.Hex())
	if err != nil {
		return fmt.Errorf("failed to impersonate account %s: %w", address.Hex(), err)
	}
	return nil
}

// StopImpersonatingAccount disables impersonation of an account on Anvil
func StopImpersonatingAccount(client *rpc.Client, address common.Address) error {
	var result interface{}
	err := client.Call(&result, "anvil_stopImpersonatingAccount", address.Hex())
	if err != nil {
		return fmt.Errorf("failed to stop impersonating account %s: %w", address.Hex(), err)
	}
	return nil
}

func (cc *ContractCaller) GetOperatorRegistrationMessageHash(
	ctx context.Context,
	operatorAddress common.Address,
	avsAddress common.Address,
	operatorSetId uint32,
	keyData []byte,
) ([32]byte, error) {
	keyRegitrar, err := IKeyRegistrar.NewIKeyRegistrar(cc.keyRegistrarAddr, cc.ethclient)
	if err != nil {
		return [32]byte{}, fmt.Errorf("failed to create key registrar contract: %w", err)
	}
	return keyRegitrar.GetBN254KeyRegistrationMessageHash(&bind.CallOpts{Context: ctx}, operatorAddress, IKeyRegistrar.OperatorSet{
		Avs: avsAddress,
		Id:  operatorSetId,
	}, keyData)
}

func (cc *ContractCaller) EncodeBN254KeyData(pubKey *bn254.PublicKey) ([]byte, error) {
	// Convert G1 point
	g1Point := &bn254.G1Point{
		G1Affine: pubKey.GetG1Point(),
	}
	g1Bytes, err := g1Point.ToPrecompileFormat()
	if err != nil {
		return nil, fmt.Errorf("public key not in correct subgroup: %w", err)
	}

	keyRegG1 := IKeyRegistrar.BN254G1Point{
		X: new(big.Int).SetBytes(g1Bytes[0:32]),
		Y: new(big.Int).SetBytes(g1Bytes[32:64]),
	}

	g2Point := bn254.NewZeroG2Point().AddPublicKey(pubKey)
	g2Bytes, err := g2Point.ToPrecompileFormat()
	if err != nil {
		return nil, fmt.Errorf("public key not in correct subgroup: %w", err)
	}
	// Convert to IKeyRegistrar G2 point format
	keyRegG2 := IKeyRegistrar.BN254G2Point{
		X: [2]*big.Int{
			new(big.Int).SetBytes(g2Bytes[0:32]),
			new(big.Int).SetBytes(g2Bytes[32:64]),
		},
		Y: [2]*big.Int{
			new(big.Int).SetBytes(g2Bytes[64:96]),
			new(big.Int).SetBytes(g2Bytes[96:128]),
		},
	}

	log.Printf("keyRegistrarAddr: %s", cc.keyRegistrarAddr)
	keyRegistrarContract, err := IKeyRegistrar.NewIKeyRegistrar(cc.keyRegistrarAddr, cc.ethclient)
	if err != nil {
		return nil, fmt.Errorf("failed to create key registrar contract: %w", err)
	}
	return keyRegistrarContract.EncodeBN254KeyData(&bind.CallOpts{}, keyRegG1, keyRegG2)
}

// IsSemver checks if a version string is valid
func IsSemver(s string) bool {
	return semverRegex.MatchString(s)
}

// ParseVersion converts version string like "0.0.5" to comparable integers
func ParseVersion(v string) (major, minor, patch int, err error) {
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %s", v)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return major, minor, patch, nil
}

// CompareVersions returns true if v1 > v2
func CompareVersions(v1, v2 string) (bool, error) {
	major1, minor1, patch1, err := ParseVersion(v1)
	if err != nil {
		return false, fmt.Errorf("parse version %s: %w", v1, err)
	}

	major2, minor2, patch2, err := ParseVersion(v2)
	if err != nil {
		return false, fmt.Errorf("parse version %s: %w", v2, err)
	}

	if major1 > major2 {
		return true, nil
	}
	if major1 < major2 {
		return false, nil
	}

	if minor1 > minor2 {
		return true, nil
	}
	if minor1 < minor2 {
		return false, nil
	}

	return patch1 > patch2, nil
}
