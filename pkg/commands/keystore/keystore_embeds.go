package keystore

import (
	"embed"
)

//go:embed create_keystore.go keystore.go
var files embed.FS

var KeystoreEmbeds map[string]string

func init() {
	// copy everything over except for the embeds
	createKeystoreBytes, _ := files.ReadFile("create_keystore.go")
	keystoreBytes, _ := files.ReadFile("keystore.go")
	keystoreTestBytes, _ := files.ReadFile("keystore_test.go")
	readKeystoreBytes, _ := files.ReadFile("read_keystore.go")

	KeystoreEmbeds = map[string]string{
		"create_keystore.go": string(createKeystoreBytes),
		"keystore.go":        string(keystoreBytes),
		"keystore_test.go":   string(keystoreTestBytes),
		"read_keystore.go":   string(readKeystoreBytes),
	}
}
