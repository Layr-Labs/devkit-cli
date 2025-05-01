package keystore

import (
	"devkit-cli/pkg/common"
	"fmt"
	"log"

	// "encoding/hex"
	"github.com/Layr-Labs/hourglass-monorepo/ponos/pkg/signing/bn254"
	"github.com/Layr-Labs/hourglass-monorepo/ponos/pkg/signing/keystore"
	"github.com/urfave/cli/v2"
)

var ReadCommand = &cli.Command{
	Name:  "read",
	Usage: "Read the bls key from a given keystore file, password",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:     "path",
			Usage:    "Path to the keystore JSON",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "type",
			Usage:    "Curve type (only 'bn254' supported)",
			Required: false,
		},
		&cli.StringFlag{
			Name:     "password",
			Usage:    "Password to decrypt the keystore file",
			Required: false,
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		path := cCtx.String("path")
		curve := cCtx.String("type")
		password := cCtx.String("password")

		log.Printf("🔐 Creating keystore with curve '%s' at: %s", curve, path)
		scheme := bn254.NewScheme()
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
