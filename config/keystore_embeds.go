package config

import _ "embed"

//go:embed keystores/operator1.keystore.json
var operator1Keystore string

//go:embed keystores/operator2.keystore.json
var operator2Keystore string

//go:embed keystores/operator3.keystore.json
var operator3Keystore string

//go:embed keystores/operator4.keystore.json
var operator4Keystore string

//go:embed keystores/operator5.keystore.json
var operator5Keystore string

// Map of context name → content
var KeystoreEmbeds = map[string]string{
	"operator1.keystore.json": operator1Keystore,
	"operator2.keystore.json": operator2Keystore,
	"operator3.keystore.json": operator3Keystore,
	"operator4.keystore.json": operator4Keystore,
	"operator5.keystore.json": operator5Keystore,
}
