package keystore

import (
	"devkit-cli/pkg/common"
	"fmt"
	"log"
	"path/filepath"

	"github.com/Layr-Labs/hourglass-monorepo/ponos/pkg/signing/bn254"
	"github.com/Layr-Labs/hourglass-monorepo/ponos/pkg/signing/keystore"
	"github.com/urfave/cli/v2"
)

var CreateCommand = &cli.Command{
	Name:  "create",
	Usage: "Generates a BLS keystore JSON file for a private key",
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
		privateKey := cCtx.String("key")
		path := cCtx.String("path")
		curve := cCtx.String("type")
		password := cCtx.String("password")
		verbose := cCtx.Bool("verbose")

		if path == "" || len(path) < 6 || filepath.Ext(path) != ".json" {
			return fmt.Errorf("invalid path: must include full file name ending in .json")
		}

		if verbose {
			log.Printf("🔐 Starting Bls keystore creation")
			log.Printf("• Curve: %s", curve)
			log.Printf("• Output Path: %s", path)
		}

		scheme := bn254.NewScheme()
		ke, err := scheme.NewPrivateKeyFromBytes([]byte(privateKey))
		if err != nil {
			return fmt.Errorf("failed to create private key from bytes: %w", err)
		}

		err = keystore.SaveToKeystoreWithCurveType(ke, path, password, curve, keystore.Default())
		if err != nil {
			return fmt.Errorf("failed to create keystore: %w", err)
		}

		keystoreData, _ := keystore.LoadKeystoreFile(path)

		privateKeyData, err := keystoreData.GetPrivateKey(password, scheme)
		if err != nil {
			return fmt.Errorf("failed to extract the private key from the keystore file")
		}
		log.Println("✅ Keystore generated successfully")
		log.Println("")
		log.Println("🔑 Save this BLS private key in a secure location:")
		log.Printf("    %s\n", privateKeyData.Bytes())
		log.Println("")

		return nil
	},
}
