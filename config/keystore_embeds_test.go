package config

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// KeystoreJSON represents the structure of a keystore file
type KeystoreJSON struct {
	Address string                 `json:"address"`
	Crypto  map[string]interface{} `json:"crypto"`
	ID      string                 `json:"id"`
	Version int                    `json:"version"`
}

func TestEmbeddedECDSAKeystores(t *testing.T) {
	// List of expected ECDSA keystore files
	expectedECDSAKeystores := []string{
		"operator1.ecdsa.keystore.json",
		"operator2.ecdsa.keystore.json",
		"operator3.ecdsa.keystore.json",
		"operator4.ecdsa.keystore.json",
		"operator5.ecdsa.keystore.json",
	}

	// Verify each ECDSA keystore exists and is valid
	for _, filename := range expectedECDSAKeystores {
		t.Run(filename, func(t *testing.T) {
			content, exists := KeystoreEmbeds[filename]
			require.True(t, exists, "ECDSA keystore %s should be embedded", filename)
			require.NotEmpty(t, content, "ECDSA keystore %s should not be empty", filename)

			// Verify it's valid JSON
			var keystore KeystoreJSON
			err := json.Unmarshal([]byte(content), &keystore)
			require.NoError(t, err, "ECDSA keystore %s should be valid JSON", filename)

			// Verify keystore structure
			require.NotEmpty(t, keystore.Address, "ECDSA keystore should have an address")
			require.NotEmpty(t, keystore.Crypto, "ECDSA keystore should have crypto section")
			require.NotEmpty(t, keystore.ID, "ECDSA keystore should have an ID")
			require.Equal(t, 3, keystore.Version, "ECDSA keystore should be version 3")

			// Verify crypto section has required fields
			require.Contains(t, keystore.Crypto, "cipher")
			require.Contains(t, keystore.Crypto, "ciphertext")
			require.Contains(t, keystore.Crypto, "cipherparams")
			require.Contains(t, keystore.Crypto, "kdf")
			require.Contains(t, keystore.Crypto, "kdfparams")
			require.Contains(t, keystore.Crypto, "mac")
		})
	}
}

func TestEmbeddedBLSKeystores(t *testing.T) {
	// List of expected BLS keystore files
	expectedBLSKeystores := []string{
		"operator1.bls.keystore.json",
		"operator2.bls.keystore.json",
		"operator3.bls.keystore.json",
		"operator4.bls.keystore.json",
		"operator5.bls.keystore.json",
	}

	// Verify each BLS keystore exists and is valid
	for _, filename := range expectedBLSKeystores {
		t.Run(filename, func(t *testing.T) {
			content, exists := KeystoreEmbeds[filename]
			require.True(t, exists, "BLS keystore %s should be embedded", filename)
			require.NotEmpty(t, content, "BLS keystore %s should not be empty", filename)

			// Verify it's valid JSON
			var keystore map[string]interface{}
			err := json.Unmarshal([]byte(content), &keystore)
			require.NoError(t, err, "BLS keystore %s should be valid JSON", filename)

			// BLS keystores have a different structure
			require.Contains(t, keystore, "pubkey", "BLS keystore should have pubkey")
			require.Contains(t, keystore, "crypto", "BLS keystore should have crypto section")
			require.Contains(t, keystore, "version", "BLS keystore should have version")
		})
	}
}

func TestKeystoreNamingConvention(t *testing.T) {
	// Verify all keystores follow the correct naming convention
	for filename := range KeystoreEmbeds {
		t.Run(filename, func(t *testing.T) {
			require.True(t, strings.HasSuffix(filename, ".keystore.json"), 
				"Keystore file %s should end with .keystore.json", filename)
			
			if strings.Contains(filename, ".ecdsa.") {
				// ECDSA keystore naming convention
				require.Regexp(t, `^operator\d+\.ecdsa\.keystore\.json$`, filename,
					"ECDSA keystore %s should follow naming convention", filename)
			} else if strings.Contains(filename, ".bls.") {
				// BLS keystore naming convention
				require.Regexp(t, `^operator\d+\.bls\.keystore\.json$`, filename,
					"BLS keystore %s should follow naming convention", filename)
			} else {
				t.Errorf("Keystore %s doesn't follow either ECDSA or BLS naming convention", filename)
			}
		})
	}
}

func TestAllOperatorsHaveBothKeystoreTypes(t *testing.T) {
	// Verify each operator has both ECDSA and BLS keystores
	for i := 1; i <= 5; i++ {
		ecdsaFile := "operator" + string(rune('0'+i)) + ".ecdsa.keystore.json"
		blsFile := "operator" + string(rune('0'+i)) + ".bls.keystore.json"

		t.Run("operator"+string(rune('0'+i)), func(t *testing.T) {
			_, hasECDSA := KeystoreEmbeds[ecdsaFile]
			_, hasBLS := KeystoreEmbeds[blsFile]
			
			require.True(t, hasECDSA, "Operator %d should have ECDSA keystore", i)
			require.True(t, hasBLS, "Operator %d should have BLS keystore", i)
		})
	}
}

func TestECDSAKeystoreAddresses(t *testing.T) {
	// Expected addresses for each operator (without 0x prefix, lowercase)
	expectedAddresses := map[string]string{
		"operator1.ecdsa.keystore.json": "90f79bf6eb2c4f870365e785982e1f101e93b906",
		"operator2.ecdsa.keystore.json": "15d34aaf54267db7d7c367839aaf71a00a2c6a65",
		"operator3.ecdsa.keystore.json": "9965507d1a55bcc2695c58ba16fb37d819b0a4dc",
		"operator4.ecdsa.keystore.json": "976ea74026e726554db657fa54763abd0c3a0aa9",
		"operator5.ecdsa.keystore.json": "14dc79964da2c08b23698b3d3cc7ca32193d9955",
	}

	for filename, expectedAddr := range expectedAddresses {
		t.Run(filename, func(t *testing.T) {
			content, exists := KeystoreEmbeds[filename]
			require.True(t, exists)

			var keystore KeystoreJSON
			err := json.Unmarshal([]byte(content), &keystore)
			require.NoError(t, err)

			// Keystore addresses are stored without 0x prefix and in lowercase
			require.Equal(t, expectedAddr, strings.ToLower(keystore.Address),
				"ECDSA keystore %s should have correct address", filename)
		})
	}
}