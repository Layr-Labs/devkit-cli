package keystore

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"

	"github.com/Layr-Labs/hourglass-monorepo/ponos/pkg/signing/bn254"
	"github.com/Layr-Labs/hourglass-monorepo/ponos/pkg/signing/keystore"
	"github.com/urfave/cli/v2"
)

var CreateCommand = &cli.Command{
	Name:  "create",
	Usage: "Generates a Bls keystore JSON file for a private key",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:     "key",
			Usage:    "Bls private key in large number",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "path",
			Usage:    "Full path to save keystore file, including filename (e.g., ./operator_keys/operator1.json)",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "type",
			Usage: "Curve type (only 'bn254' supported)",
			Value: "bn254",
		},
		&cli.StringFlag{
			Name:  "password",
			Usage: `Password to encrypt the keystore file. Default password is "" `,
			Value: "",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx.Context)

		privateKey := cCtx.String("key")
		path := cCtx.String("path")
		curve := cCtx.String("type")
		password := cCtx.String("password")

		logger.Debug("🔐 Starting Bls keystore creation")
		logger.Debug("• Curve: %s", curve)
		logger.Debug("• Output Path: %s", path)

		return CreateBLSKeystore(logger, privateKey, path, password, curve)
	},
}

func CreateBLSKeystore(logger iface.Logger, privateKey, path, password, curve string) error {

	if filepath.Ext(path) != ".json" {
		return errors.New("invalid path: must include full file name ending in .json")
	}

	if curve != "bn254" {
		return fmt.Errorf("unsupported curve: %s", curve)
	}

	logger.Debug("🔐 Starting Bls keystore creation")
	logger.Debug("• Curve: %s", curve)
	logger.Debug("• Output Path: %s", path)

	scheme := bn254.NewScheme()
	cleanedKey := strings.TrimPrefix(privateKey, "0x")
	ke, err := scheme.NewPrivateKeyFromBytes([]byte(cleanedKey))
	if err != nil {
		return fmt.Errorf("failed to create private key from bytes: %w", err)
	}

	err = keystore.SaveToKeystoreWithCurveType(ke, path, password, curve, keystore.Default())
	if err != nil {
		return fmt.Errorf("failed to create keystore: %w", err)
	}

	keystoreData, err := keystore.LoadKeystoreFile(path)
	if err != nil {
		return fmt.Errorf("failed to reload keystore: %w", err)
	}

	privateKeyData, err := keystoreData.GetPrivateKey(password, scheme)
	if err != nil {
		return errors.New("failed to extract the private key from the keystore file")
	}

	logger.Info("✅ Keystore generated successfully")
	logger.Info("🔑 Save this BLS private key in a secure location:")
	logger.Info("    %s\n", privateKeyData.Bytes())

	return nil
}
