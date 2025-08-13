package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Layr-Labs/devkit-cli/pkg/common"
	"github.com/stretchr/testify/require"
)

func TestLoadECDSAKeysFromKeystores(t *testing.T) {
	// Test operator configurations
	operators := []common.OperatorSpec{
		{
			Address:  "0x90F79bf6EB2c4f870365E785982E1f101E93b906",
			ECDSAKey: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
			// No keystore - use plaintext
		},
		{
			Address:  "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65",
			ECDSAKey: "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
			Keystores: []common.OperatorKeystores{
				{
					ECDSAKeystorePath:     "keystores/nonexistent.ecdsa.keystore.json",
					ECDSAKeystorePassword: "testpass",
				},
			},
		},
		{
			Address:  "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc",
			ECDSAKey: "0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba",
			// No keystore paths - should use plaintext key
		},
	}

	t.Run("use plaintext key when no keystore specified", func(t *testing.T) {
		// Test first operator with no keystore
		key, err := loadOperatorECDSAKey(operators[0])
		require.NoError(t, err)
		require.Equal(t, "7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6", key)
	})

	t.Run("error when keystore file not found", func(t *testing.T) {
		// Test second operator with non-existent keystore file
		_, err := loadOperatorECDSAKey(operators[1])
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read ECDSA keystore file")
	})

	t.Run("use plaintext key when no keystore specified 2", func(t *testing.T) {
		// Test third operator with no keystore
		key, err := loadOperatorECDSAKey(operators[2])
		require.NoError(t, err)
		require.Equal(t, "8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba", key)
	})

	t.Run("error when no key available", func(t *testing.T) {
		// Test operator with no key at all
		emptyOp := common.OperatorSpec{
			Address: "0x14dC79964da2C08b23698B3D3cc7Ca32193d9955",
		}
		_, err := loadOperatorECDSAKey(emptyOp)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no ECDSA key configuration found")
	})

	t.Run("use plaintext key when keystore path without password", func(t *testing.T) {
		// Test operator with keystore path but no password - should use plaintext
		opWithPath := common.OperatorSpec{
			Address:  "0x14dC79964da2C08b23698B3D3cc7Ca32193d9955",
			ECDSAKey: "0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356",
			Keystores: []common.OperatorKeystores{
				{
					ECDSAKeystorePath: "keystores/operator5.ecdsa.keystore.json",
					// No password provided
				},
			},
		}
		key, err := loadOperatorECDSAKey(opWithPath)
		require.NoError(t, err)
		require.Equal(t, "4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356", key)
	})
}

func TestMixedBLSAndECDSAOperators(t *testing.T) {
	// Create test directory and keystore files
	tmpDir := t.TempDir()
	keystoreDir := filepath.Join(tmpDir, "keystores")
	err := os.MkdirAll(keystoreDir, 0755)
	require.NoError(t, err)

	// Change to test directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func(dir string) {
		err = os.Chdir(dir)
		if err != nil {
			t.Fail()
		}
	}(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Test mixed operator configuration
	operators := []common.OperatorSpec{
		{
			Address:  "0x90F79bf6EB2c4f870365E785982E1f101E93b906",
			ECDSAKey: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
			Keystores: []common.OperatorKeystores{
				{
					ECDSAKeystorePath:     "keystores/operator1.ecdsa.keystore.json",
					ECDSAKeystorePassword: "testpass",
					BlsKeystorePath:       "keystores/operator1.bls.keystore.json",
					BlsKeystorePassword:   "testpass",
				},
			},
		},
		{
			Address:  "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65",
			ECDSAKey: "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a",
			Keystores: []common.OperatorKeystores{
				{
					BlsKeystorePath:     "keystores/operator2.bls.keystore.json",
					BlsKeystorePassword: "testpass",
				},
			},

			// No ECDSA keystore - uses plaintext
		},
	}

	// Use the actual embedded ECDSA keystore content (encrypted with "testpass")
	ecdsaKeystoreContent := `{
  "address": "90f79bf6eb2c4f870365e785982e1f101e93b906",
  "crypto": {
    "cipher": "aes-128-ctr",
    "ciphertext": "cb9721b2bb12f67f63eadcbc42e7a2ac5721b03dbe50d07c723cd7693921b477",
    "cipherparams": {
      "iv": "d3395f83b964967800b3071195cee867"
    },
    "kdf": "scrypt",
    "kdfparams": {
      "dklen": 32,
      "n": 262144,
      "p": 1,
      "r": 8,
      "salt": "0fe7be44e9d127275d57a0e841be14f0e722467d69136f28683c3a1ad8c98333"
    },
    "mac": "d43b0060ac1e5a95f66fc99493bf7bb0c180eee12aadfe389e5706b8bda42c64"
  },
  "id": "4df14231-53b3-49c4-8d29-823197cb217c",
  "version": 3
}`

	// Create BLS keystore (simplified for testing)
	blsKeystoreContent := `{
		"pubkey": "b3a5a44586640de0a2f67ab0c47036f1e5cb9e5b4969b09c73b99ad1c48c60d996965b343c0fb417c7f08033ad30abb3",
		"crypto": {
			"cipher": "aes-128-ctr",
			"ciphertext": "76f10c0b968f2cf96e22b45aa84c97a4f9e6ccb4bbaa2b48b593de64cc829e49",
			"cipherparams": {
				"iv": "87eff5f0f9b68d90a73efd4b8e8cf913"
			},
			"kdf": "scrypt",
			"kdfparams": {
				"dklen": 32,
				"n": 8192,
				"p": 1,
				"r": 8,
				"salt": "f5758e95db38cf491cbab3f1a7f90e646c1fd688c7e17d6cdc80695fbc06bb4f"
			},
			"mac": "c436dbf0b87a007bb30ba6dd96ad0de6b0c5a87ba9b84c27de088c0f8fe7f77f"
		},
		"version": 3
	}`

	// Write keystores
	err = os.WriteFile(filepath.Join(keystoreDir, "operator1.ecdsa.keystore.json"), []byte(ecdsaKeystoreContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(keystoreDir, "operator1.bls.keystore.json"), []byte(blsKeystoreContent), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(keystoreDir, "operator2.bls.keystore.json"), []byte(blsKeystoreContent), 0644)
	require.NoError(t, err)

	t.Run("operator with both ECDSA and BLS keystores", func(t *testing.T) {
		// Test first operator can load ECDSA key from keystore
		ecdsaKey, err := loadOperatorECDSAKey(operators[0])
		require.NoError(t, err)
		require.Equal(t, "7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6", ecdsaKey)

		// Verify BLS keystore also exists
		_, err = os.Stat(filepath.Join(keystoreDir, "operator1.bls.keystore.json"))
		require.NoError(t, err)
	})

	t.Run("operator with BLS keystore and plaintext ECDSA", func(t *testing.T) {
		// Test second operator uses plaintext ECDSA
		ecdsaKey, err := loadOperatorECDSAKey(operators[1])
		require.NoError(t, err)
		require.Equal(t, "47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a", ecdsaKey)

		// Verify BLS keystore exists
		_, err = os.Stat(filepath.Join(keystoreDir, "operator2.bls.keystore.json"))
		require.NoError(t, err)
	})
}

func TestECDSAKeyNormalization(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "key with 0x prefix",
			input:    "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
			expected: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
		},
		{
			name:     "key without 0x prefix",
			input:    "7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
			expected: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
		},
		{
			name:     "uppercase key",
			input:    "0x7C852118294E51E653712A81E05800F419141751BE58F605C371E15141B007A6",
			expected: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := common.OperatorSpec{
				Address:  "0x90F79bf6EB2c4f870365E785982E1f101E93b906",
				ECDSAKey: tt.input,
			}

			key, err := loadOperatorECDSAKey(op)
			require.NoError(t, err)
			// The function returns without 0x prefix
			require.Equal(t, strings.TrimPrefix(tt.expected, "0x"), key)
		})
	}
}
