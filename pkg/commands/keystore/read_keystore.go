package keystore

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Layr-Labs/crypto-libs/pkg/bn254"
	blskeystore "github.com/Layr-Labs/crypto-libs/pkg/keystore"
	"github.com/Layr-Labs/devkit-cli/pkg/common"
	ethkeystore "github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/urfave/cli/v2"
	"log"
)

var ReadCommand = &cli.Command{
	Name:  "read",
	Usage: "Print the private key from a given keystore file, password",
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

		// Determine keystore type by checking the file content
		fileContent, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read keystore file: %w", err)
		}

		// Check if it's an ECDSA keystore (has "address" field)
		var jsonData map[string]interface{}
		if err := json.Unmarshal(fileContent, &jsonData); err != nil {
			return fmt.Errorf("failed to parse keystore JSON: %w", err)
		}

		if _, hasAddress := jsonData["address"]; hasAddress {
			// ECDSA keystore
			key, err := ethkeystore.DecryptKey(fileContent, password)
			if err != nil {
				return fmt.Errorf("failed to decrypt ECDSA keystore: %w", err)
			}

			privateKeyHex := hex.EncodeToString(key.PrivateKey.D.Bytes())
			log.Println("✅ ECDSA Keystore decrypted successfully")
			log.Println("")
			log.Println("🔑 Save this ECDSA private key in a secure location:")
			log.Printf("    0x%s\n", privateKeyHex)
			log.Println("")
		} else if _, hasPubkey := jsonData["pubkey"]; hasPubkey {
			// BLS keystore
			scheme := bn254.NewScheme()
			keystoreData, err := blskeystore.LoadKeystoreFile(path)
			if err != nil {
				return fmt.Errorf("failed to load BLS keystore file: %w", err)
			}

			privateKeyData, err := keystoreData.GetPrivateKey(password, scheme)
			if err != nil {
				return fmt.Errorf("failed to extract BLS private key: %w", err)
			}
			log.Println("✅ BLS Keystore decrypted successfully")
			log.Println("")
			log.Println("🔑 Save this BLS private key in a secure location:")
			log.Printf("    %s\n", privateKeyData.Bytes())
			log.Println("")
		} else {
			return fmt.Errorf("unknown keystore format")
		}

		return nil
	},
}
