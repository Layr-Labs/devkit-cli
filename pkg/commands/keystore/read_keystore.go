package keystore

import (
	"fmt"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/hourglass-monorepo/ponos/pkg/signing/bn254"
	"github.com/Layr-Labs/hourglass-monorepo/ponos/pkg/signing/keystore"
	"github.com/urfave/cli/v2"
	"log"
)

var ReadCommand = &cli.Command{
	Name:  "read",
	Usage: "Print the Bls key from a given keystore file, password",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:     "path",
			Usage:    "Path to the keystore JSON",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "password",
			Usage:    "Password to decrypt the keystore file",
			Required: true,
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		path := cCtx.String("path")
		password := cCtx.String("password")

		scheme := bn254.NewScheme()
		keystoreData, err := keystore.LoadKeystoreFile(path)

		if err != nil {
			return fmt.Errorf("failed to load the keystore file from given path %s", path)
		}

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
