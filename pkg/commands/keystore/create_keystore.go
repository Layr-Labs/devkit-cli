package keystore

import (
	"errors"
	"fmt"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"path/filepath"
	"strings"

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
		log, _ := common.GetLogger(cCtx.Bool("verbose"))

		privateKey := cCtx.String("key")
		path := cCtx.String("path")
		curve := cCtx.String("type")
		password := cCtx.String("password")
		verbose := cCtx.Bool("verbose")

		if verbose {
			log.Info("🔐 Starting Bls keystore creation")
			log.Info("• Curve: %s", curve)
			log.Info("• Output Path: %s", path)
		}

		return CreateBLSKeystore(privateKey, path, password, curve, verbose)
	},
}

func CreateBLSKeystore(privateKey, path, password, curve string, verbose bool) error {
	log, _ := common.GetLogger(verbose)

	if filepath.Ext(path) != ".json" {
		return errors.New("invalid path: must include full file name ending in .json")
	}

	if curve != "bn254" {
		return fmt.Errorf("unsupported curve: %s", curve)
	}

	if verbose {
		log.Info("🔐 Starting Bls keystore creation")
		log.Info("• Curve: %s", curve)
		log.Info("• Output Path: %s", path)
	}

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

	log.Info("✅ Keystore generated successfully")
	log.Info("🔑 Save this BLS private key in a secure location:")
	log.Info("    %s\n", privateKeyData.Bytes())

	return nil
}
