package keystore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/Layr-Labs/devkit-cli/pkg/common/iface"

	"github.com/Layr-Labs/crypto-libs/pkg/bn254"
	blskeystore "github.com/Layr-Labs/crypto-libs/pkg/keystore"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

var CreateCommand = &cli.Command{
	Name:  "create",
	Usage: "Generates a BLS or ECDSA keystore JSON file for a private key",
	Flags: append([]cli.Flag{
		&cli.StringFlag{
			Name:     "key",
			Usage:    "Private key (BLS private key in large number format or ECDSA private key in hex format)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "path",
			Usage:    "Full path to save keystore file, including filename (e.g., ./operator_keys/operator1.json)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "type",
			Usage:    "Curve type ('bn254' for BLS or 'ecdsa' for ECDSA)",
			Required: true,
		},
		&cli.StringFlag{
			Name:  "password",
			Usage: `Password to encrypt the keystore file. Default password is "" `,
			Value: "",
		},
	}, common.GlobalFlags...),
	Action: func(cCtx *cli.Context) error {
		logger := common.LoggerFromContext(cCtx)

		privateKey := cCtx.String("key")
		path := cCtx.String("path")
		curve := cCtx.String("type")
		password := cCtx.String("password")

		logger.Debug("🔐 Starting keystore creation")
		logger.Debug("• Curve: %s", curve)
		logger.Debug("• Output Path: %s", path)

		switch curve {
		case "bn254":
			return CreateBLSKeystore(logger, privateKey, path, password, curve)
		case "ecdsa":
			return CreateECDSAKeystore(logger, privateKey, path, password)
		default:
			return fmt.Errorf("unsupported curve type: %s (supported: bn254, ecdsa)", curve)
		}
	},
}

func CreateBLSKeystore(logger iface.Logger, privateKey, path, password, curve string) error {

	if filepath.Ext(path) != ".json" {
		return errors.New("invalid path: must include full file name ending in .json")
	}

	logger.Debug("🔐 Starting BLS keystore creation")
	logger.Debug("• Curve: %s", curve)
	logger.Debug("• Output Path: %s", path)

	scheme := bn254.NewScheme()
	cleanedKey := strings.TrimPrefix(privateKey, "0x")
	ke, err := scheme.NewPrivateKeyFromBytes([]byte(cleanedKey))
	if err != nil {
		return fmt.Errorf("failed to create private key from bytes: %w", err)
	}

	err = blskeystore.SaveToKeystoreWithCurveType(ke, path, password, curve, blskeystore.Default())
	if err != nil {
		return fmt.Errorf("failed to create keystore: %w", err)
	}

	keystoreData, err := blskeystore.LoadKeystoreFile(path)
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

func CreateECDSAKeystore(logger iface.Logger, privateKeyHex, path, password string) error {
	if filepath.Ext(path) != ".json" {
		return errors.New("invalid path: must include full file name ending in .json")
	}

	logger.Debug("🔐 Starting ECDSA keystore creation")
	logger.Debug("• Output Path: %s", path)

	// Clean the private key hex string
	cleanedKey := strings.TrimPrefix(privateKeyHex, "0x")

	// Convert hex string to ECDSA private key
	privateKey, err := crypto.HexToECDSA(cleanedKey)
	if err != nil {
		return fmt.Errorf("failed to parse ECDSA private key: %w", err)
	}

	// Get the address from the private key
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create a new keystore Key structure
	key := &keystore.Key{
		Id:         uuid.New(),
		Address:    address,
		PrivateKey: privateKey,
	}

	// Encrypt the key
	keyjson, err := keystore.EncryptKey(key, password, keystore.StandardScryptN, keystore.StandardScryptP)
	if err != nil {
		return fmt.Errorf("failed to encrypt ECDSA key: %w", err)
	}

	// Write the encrypted key to file
	if err := os.WriteFile(path, keyjson, 0600); err != nil {
		return fmt.Errorf("failed to write keystore file: %w", err)
	}

	// Validate by trying to decrypt
	decryptedKey, err := keystore.DecryptKey(keyjson, password)
	if err != nil {
		return fmt.Errorf("failed to validate keystore (decrypt failed): %w", err)
	}

	// Verify the address matches
	if decryptedKey.Address != address {
		return errors.New("keystore validation failed: address mismatch")
	}

	logger.Info("✅ ECDSA Keystore generated successfully")
	logger.Info("📍 Address: %s", address.Hex())
	logger.Info("📁 Keystore saved to: %s", path)

	return nil
}
